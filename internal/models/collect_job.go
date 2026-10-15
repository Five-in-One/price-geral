package models

import "time"

// CollectJob statuses. A job starts queued, and the bot's collect.finished
// event moves it to succeeded or failed.
const (
	CollectJobQueued    = "queued"
	CollectJobRunning   = "running"
	CollectJobSucceeded = "succeeded"
	CollectJobFailed    = "failed"
)

// CollectJob is one asynchronous run of the bot over a single storefront
// link of a market. Counters and Error stay nil until the job finishes.
type CollectJob struct {
	ID                string     `json:"id"`
	MarketID          string     `json:"market_id"`
	URL               string     `json:"url"`
	Status            string     `json:"status"`
	BlocksFound       *int       `json:"blocks_found,omitempty"`
	ProductsExtracted *int       `json:"products_extracted,omitempty"`
	ProductsInserted  *int       `json:"products_inserted,omitempty"`
	Error             *string    `json:"error,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	FinishedAt        *time.Time `json:"finished_at,omitempty"`
}
