-- Supports the two new write paths: the Admin panel's "coleta sob demanda"
-- (bot-collected) and the Partner Company page (manually registered), on
-- top of the existing seeded demo data.
ALTER TABLE markets ADD COLUMN source text NOT NULL DEFAULT 'seed';
ALTER TABLE markets ADD CONSTRAINT markets_source_check CHECK (source IN ('seed', 'company', 'bot'));

-- Lets POST /api/markets upsert-by-name: resubmitting the same market name
-- reuses the existing row instead of erroring or duplicating.
CREATE UNIQUE INDEX idx_markets_name_lower ON markets (lower(name));

ALTER TABLE products ADD COLUMN source text NOT NULL DEFAULT 'seed';
ALTER TABLE products ADD CONSTRAINT products_source_check CHECK (source IN ('seed', 'company', 'bot'));

-- A product name is only unique per market it was registered at — the same
-- name from two different markets stays two separate rows (matching a
-- product across markets/companies is a curation problem, not something
-- this endpoint decides on its own). Resubmitting the same market+name
-- upserts (updates unit/category/price) instead of duplicating.
CREATE UNIQUE INDEX idx_products_market_name_lower ON products (default_market_id, lower(name));
