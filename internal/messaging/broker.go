package messaging

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Publisher sends an event to the broker under a routing key. The HTTP
// handlers depend on this interface so tests can swap in a fake.
type Publisher interface {
	Publish(ctx context.Context, routingKey string, event any) error
}

// Broker holds one AMQP connection, dialed lazily and redialed when it
// drops, so price-back starts (and serves every other route) even while
// RabbitMQ is still coming up.
type Broker struct {
	url string

	mu   sync.Mutex
	conn *amqp.Connection
}

func NewBroker(url string) *Broker {
	return &Broker{url: url}
}

// connection returns the live connection, dialing a new one if needed, and
// makes sure Exchange exists.
func (b *Broker) connection() (*amqp.Connection, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.conn != nil && !b.conn.IsClosed() {
		return b.conn, nil
	}

	conn, err := amqp.Dial(b.url)
	if err != nil {
		return nil, fmt.Errorf("dial rabbitmq: %w", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("open channel: %w", err)
	}
	defer ch.Close()
	if err := ch.ExchangeDeclare(Exchange, amqp.ExchangeTopic, true, false, false, false, nil); err != nil {
		conn.Close()
		return nil, fmt.Errorf("declare exchange %s: %w", Exchange, err)
	}

	b.conn = conn
	return conn, nil
}

// Publish serializes event as JSON and publishes it as a persistent
// message to Exchange, waiting for the broker's confirmation so a 202 is
// only returned once the event is safely stored.
func (b *Broker) Publish(ctx context.Context, routingKey string, event any) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	conn, err := b.connection()
	if err != nil {
		return err
	}
	// A channel per publish keeps this safe across concurrent requests;
	// the volume here (one per collection) makes the cost irrelevant.
	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}
	defer ch.Close()
	if err := ch.Confirm(false); err != nil {
		return fmt.Errorf("enable publisher confirms: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	confirm, err := ch.PublishWithDeferredConfirmWithContext(ctx, Exchange, routingKey, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		MessageId:    NewEventID(),
		Timestamp:    time.Now().UTC(),
		Body:         body,
	})
	if err != nil {
		return fmt.Errorf("publish %s: %w", routingKey, err)
	}
	ok, err := confirm.WaitContext(ctx)
	if err != nil {
		return fmt.Errorf("confirm %s: %w", routingKey, err)
	}
	if !ok {
		return fmt.Errorf("broker rejected %s", routingKey)
	}
	return nil
}

// Close closes the connection, if one is open.
func (b *Broker) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.conn != nil {
		b.conn.Close()
	}
}

// NewEventID returns a random id for an event.
func NewEventID() string {
	buf := make([]byte, 16)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}
