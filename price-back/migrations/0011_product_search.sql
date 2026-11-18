-- The app's product search (GET /api/products/search) matches names ignoring
-- accents ("acucar" finds "Açúcar"), so it needs unaccent; pg_trgm is already
-- installed by 0009. The price index serves the default "cheapest first" sort.
CREATE EXTENSION IF NOT EXISTS unaccent;

CREATE INDEX IF NOT EXISTS idx_market_products_price ON market_products (price);
