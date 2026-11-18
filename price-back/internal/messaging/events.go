// Package messaging connects price-back to the RabbitMQ message broker.
//
// Every event goes to a single durable topic exchange (Exchange), keyed by
// a routing key that names the event. Publishers only know the exchange and
// the routing key — never which queues or services consume it — and each
// consumer declares and binds its own queue.
package messaging

import "time"

// Exchange is the topic exchange all Price services publish events to.
const Exchange = "price.events"

// Routing keys of the events exchanged through Exchange.
const (
	// CollectRequestedKey: a collection of one market link was requested.
	CollectRequestedKey = "collect.requested"
	// CollectFinishedKey: a bot worker finished (or failed) a collection.
	CollectFinishedKey = "collect.finished"
)

// CollectRequested is published by price-back when the Admin panel asks
// for a market link to be collected. The job is already saved as queued.
type CollectRequested struct {
	EventID     string    `json:"event_id"`
	JobID       string    `json:"job_id"`
	MarketID    string    `json:"market_id"`
	MarketName  string    `json:"market_name"`
	URL         string    `json:"url"`
	RequestedAt time.Time `json:"requested_at"`
}

// CollectFinished is published by the bot worker once a collection ends.
// Status is "succeeded" or "failed"; Error explains a failure.
type CollectFinished struct {
	EventID           string    `json:"event_id"`
	JobID             string    `json:"job_id"`
	Status            string    `json:"status"`
	BlocksFound       int       `json:"blocks_found"`
	ProductsExtracted int       `json:"products_extracted"`
	ProductsInserted  int       `json:"products_inserted"`
	Error             *string   `json:"error"`
	FinishedAt        time.Time `json:"finished_at"`
}
