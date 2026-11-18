package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"priceback/internal/models"
	"priceback/internal/repository"
)

// CollectStatusQueue is price-back's own queue for collect.finished events.
const CollectStatusQueue = "back.collect-status"

// JobFinisher closes a collect job; implemented by service.CollectService.
type JobFinisher interface {
	Finish(ctx context.Context, id string, in repository.FinishJobInput) error
}

// HandleCollectFinished decodes one collect.finished message and closes the
// matching job. It returns requeue=true only for transient errors (e.g. the
// database is down); a malformed message or an unknown job is dropped.
func HandleCollectFinished(ctx context.Context, jobs JobFinisher, body []byte) (requeue bool, err error) {
	var ev CollectFinished
	if err := json.Unmarshal(body, &ev); err != nil {
		return false, fmt.Errorf("invalid collect.finished payload: %w", err)
	}
	if ev.JobID == "" || (ev.Status != models.CollectJobSucceeded && ev.Status != models.CollectJobFailed) {
		return false, fmt.Errorf("invalid collect.finished event: job_id=%q status=%q", ev.JobID, ev.Status)
	}

	err = jobs.Finish(ctx, ev.JobID, repository.FinishJobInput{
		Status:            ev.Status,
		BlocksFound:       ev.BlocksFound,
		ProductsExtracted: ev.ProductsExtracted,
		ProductsInserted:  ev.ProductsInserted,
		Error:             ev.Error,
	})
	if errors.Is(err, repository.ErrJobNotFound) {
		return false, err
	}
	if err != nil {
		return true, err
	}
	log.Printf("[messaging] job %s finished: status=%s inserted=%d", ev.JobID, ev.Status, ev.ProductsInserted)
	return false, nil
}

// ConsumeCollectFinished listens on CollectStatusQueue until ctx is done,
// reconnecting with backoff whenever the broker is unreachable or the
// connection drops.
func ConsumeCollectFinished(ctx context.Context, b *Broker, jobs JobFinisher) {
	backoff := time.Second
	for ctx.Err() == nil {
		err := consumeOnce(ctx, b, jobs)
		if ctx.Err() != nil {
			return
		}
		log.Printf("[messaging] consumer %s stopped: %v — retrying in %s", CollectStatusQueue, err, backoff)
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func consumeOnce(ctx context.Context, b *Broker, jobs JobFinisher) error {
	conn, err := b.connection()
	if err != nil {
		return err
	}
	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}
	defer ch.Close()

	if _, err := ch.QueueDeclare(CollectStatusQueue, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare queue: %w", err)
	}
	if err := ch.QueueBind(CollectStatusQueue, CollectFinishedKey, Exchange, false, nil); err != nil {
		return fmt.Errorf("bind queue: %w", err)
	}
	if err := ch.Qos(1, 0, false); err != nil {
		return fmt.Errorf("set qos: %w", err)
	}
	deliveries, err := ch.Consume(CollectStatusQueue, "price-back", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume: %w", err)
	}

	log.Printf("[messaging] listening on queue %s (%s)", CollectStatusQueue, CollectFinishedKey)
	closed := ch.NotifyClose(make(chan *amqp.Error, 1))
	for {
		select {
		case <-ctx.Done():
			return nil
		case amqpErr := <-closed:
			return fmt.Errorf("channel closed: %v", amqpErr)
		case d, ok := <-deliveries:
			if !ok {
				return errors.New("delivery channel closed")
			}
			requeue, err := HandleCollectFinished(ctx, jobs, d.Body)
			if err != nil {
				log.Printf("[messaging] collect.finished not processed (requeue=%v): %v", requeue, err)
				_ = d.Nack(false, requeue)
				continue
			}
			_ = d.Ack(false)
		}
	}
}
