// Package service holds the business logic behind the HTTP handlers
// (controllers) and the broker consumers.
package service

import (
	"context"
	"errors"
	"log"
	"time"

	"priceback/internal/messaging"
	"priceback/internal/models"
	"priceback/internal/repository"
)

var (
	// ErrBrokerUnavailable: the collect.requested event could not be
	// published. The job is closed as failed; the market stays saved.
	ErrBrokerUnavailable = errors.New("message broker unavailable")
	// ErrNoLinks: a reload was asked for a market with no saved link.
	ErrNoLinks = errors.New("market has no links")
)

// CollectInProgressError is returned when the same market link already
// has a queued or running job; Job is that job, for the client to follow.
type CollectInProgressError struct {
	Job models.CollectJob
}

func (e *CollectInProgressError) Error() string { return "collect already in progress" }

// CollectService runs the asynchronous collection flow. On the producer
// side it saves the market and the job locally (price-back's database) and
// publishes collect.requested; it never knows who consumes the event. On
// the consumer side it closes the job from the bot's collect.finished.
type CollectService struct {
	markets   *repository.MarketsRepo
	jobs      *repository.CollectJobsRepo
	publisher messaging.Publisher
}

func NewCollectService(markets *repository.MarketsRepo, jobs *repository.CollectJobsRepo, publisher messaging.Publisher) *CollectService {
	return &CollectService{markets: markets, jobs: jobs, publisher: publisher}
}

// Enabled reports whether a broker is configured to publish to.
func (s *CollectService) Enabled() bool {
	return s.publisher != nil
}

// RequestCollect saves (or updates) the market with the link, then queues
// a collection of that link. The market is saved first so it survives a
// failed collection and can be retried with ReloadMarket.
func (s *CollectService) RequestCollect(ctx context.Context, market repository.CreateMarketInput, url string) (models.Market, models.CollectJob, error) {
	market.Link = url
	saved, err := s.markets.FindOrCreate(ctx, market)
	if err != nil {
		return models.Market{}, models.CollectJob{}, err
	}

	active, found, err := s.jobs.FindActive(ctx, saved.ID, url)
	if err != nil {
		return saved, models.CollectJob{}, err
	}
	if found {
		return saved, models.CollectJob{}, &CollectInProgressError{Job: active}
	}

	job, err := s.enqueue(ctx, saved, url)
	return saved, job, err
}

// ReloadMarket queues one collection per saved link of a market ("Recarregar").
// Links that already have a job in flight reuse it.
func (s *CollectService) ReloadMarket(ctx context.Context, marketID string) ([]models.CollectJob, error) {
	market, err := s.markets.GetByID(ctx, marketID)
	if err != nil {
		return nil, err
	}
	if len(market.Links) == 0 {
		return nil, ErrNoLinks
	}

	jobs := make([]models.CollectJob, 0, len(market.Links))
	for _, url := range market.Links {
		active, found, err := s.jobs.FindActive(ctx, market.ID, url)
		if err != nil {
			return nil, err
		}
		if found {
			jobs = append(jobs, active)
			continue
		}
		job, err := s.enqueue(ctx, market, url)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

// GetJob returns a job, or repository.ErrJobNotFound.
func (s *CollectService) GetJob(ctx context.Context, id string) (models.CollectJob, error) {
	return s.jobs.GetByID(ctx, id)
}

// Finish closes a job with the outcome the bot reported (called by the
// collect.finished consumer).
func (s *CollectService) Finish(ctx context.Context, id string, in repository.FinishJobInput) error {
	return s.jobs.Finish(ctx, id, in)
}

// enqueue creates a queued job for one market link and publishes its
// collect.requested event. If the event can't be published the job is
// closed as failed, so it doesn't sit "queued" forever.
func (s *CollectService) enqueue(ctx context.Context, market models.Market, url string) (models.CollectJob, error) {
	job, err := s.jobs.Create(ctx, market.ID, url)
	if err != nil {
		return models.CollectJob{}, err
	}

	err = s.publisher.Publish(ctx, messaging.CollectRequestedKey, messaging.CollectRequested{
		EventID:     messaging.NewEventID(),
		JobID:       job.ID,
		MarketID:    market.ID,
		MarketName:  market.Name,
		URL:         url,
		RequestedAt: time.Now().UTC(),
	})
	if err != nil {
		log.Printf("publish collect.requested for job %s failed: %v", job.ID, err)
		msg := "broker_unavailable"
		_ = s.jobs.Finish(ctx, job.ID, repository.FinishJobInput{Status: models.CollectJobFailed, Error: &msg})
		return models.CollectJob{}, ErrBrokerUnavailable
	}
	return job, nil
}
