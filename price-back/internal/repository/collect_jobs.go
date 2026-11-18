package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"priceback/internal/models"
)

var ErrJobNotFound = errors.New("collect job not found")

type CollectJobsRepo struct {
	pool *pgxpool.Pool
}

func NewCollectJobsRepo(pool *pgxpool.Pool) *CollectJobsRepo {
	return &CollectJobsRepo{pool: pool}
}

const collectJobColumns = `
	id, market_id, url, status, blocks_found, products_extracted, products_inserted,
	error, created_at, updated_at, finished_at
`

func scanCollectJob(row interface{ Scan(...any) error }, j *models.CollectJob) error {
	return row.Scan(&j.ID, &j.MarketID, &j.URL, &j.Status, &j.BlocksFound, &j.ProductsExtracted,
		&j.ProductsInserted, &j.Error, &j.CreatedAt, &j.UpdatedAt, &j.FinishedAt)
}

// Create records a new queued job for one link of a market.
func (r *CollectJobsRepo) Create(ctx context.Context, marketID, url string) (models.CollectJob, error) {
	var job models.CollectJob
	err := scanCollectJob(r.pool.QueryRow(ctx, `
		INSERT INTO collect_jobs (id, market_id, url)
		VALUES ($1, $2, $3)
		RETURNING `+collectJobColumns,
		generateID("job", "job"), marketID, url,
	), &job)
	return job, err
}

// GetByID returns a job, or ErrJobNotFound.
func (r *CollectJobsRepo) GetByID(ctx context.Context, id string) (models.CollectJob, error) {
	var job models.CollectJob
	err := scanCollectJob(r.pool.QueryRow(ctx,
		`SELECT `+collectJobColumns+` FROM collect_jobs WHERE id = $1`, id,
	), &job)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.CollectJob{}, ErrJobNotFound
	}
	return job, err
}

// FindActive returns the queued or running job for this market and URL
// (matched case-insensitively), if any — used to refuse a duplicate
// collection while one is still in flight.
func (r *CollectJobsRepo) FindActive(ctx context.Context, marketID, url string) (models.CollectJob, bool, error) {
	var job models.CollectJob
	err := scanCollectJob(r.pool.QueryRow(ctx, `
		SELECT `+collectJobColumns+` FROM collect_jobs
		WHERE market_id = $1 AND lower(url) = lower($2) AND status IN ('queued', 'running')
		ORDER BY created_at DESC
		LIMIT 1
	`, marketID, url), &job)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.CollectJob{}, false, nil
	}
	if err != nil {
		return models.CollectJob{}, false, err
	}
	return job, true, nil
}

// FinishJobInput is the outcome the bot reported for a job.
type FinishJobInput struct {
	Status            string
	BlocksFound       int
	ProductsExtracted int
	ProductsInserted  int
	Error             *string
}

// Finish closes a job with the bot's result. A job that is already
// finished is left untouched (redelivered events are harmless), and an
// unknown id returns ErrJobNotFound.
func (r *CollectJobsRepo) Finish(ctx context.Context, id string, in FinishJobInput) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE collect_jobs SET
			status             = $2,
			blocks_found       = $3,
			products_extracted = $4,
			products_inserted  = $5,
			error              = $6,
			updated_at         = now(),
			finished_at        = now()
		WHERE id = $1 AND status IN ('queued', 'running')
	`, id, in.Status, in.BlocksFound, in.ProductsExtracted, in.ProductsInserted, in.Error)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		if _, err := r.GetByID(ctx, id); err != nil {
			return err
		}
	}
	return nil
}
