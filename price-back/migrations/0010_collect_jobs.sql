-- A collection ("coleta") requested from the Admin panel no longer runs
-- inside the HTTP request: price-back records a job here, publishes a
-- collect.requested event to the message broker and answers 202 right away.
-- The bot worker that picks the event up reports back with a
-- collect.finished event, which closes the job (see internal/messaging).
CREATE TABLE collect_jobs (
    id                  text PRIMARY KEY,
    market_id           text NOT NULL REFERENCES markets(id) ON DELETE CASCADE,
    url                 text NOT NULL,
    status              text NOT NULL DEFAULT 'queued'
                        CHECK (status IN ('queued', 'running', 'succeeded', 'failed')),
    blocks_found        integer,
    products_extracted  integer,
    products_inserted   integer,
    error               text,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    finished_at         timestamptz
);

CREATE INDEX idx_collect_jobs_market_url ON collect_jobs (market_id, lower(url));
