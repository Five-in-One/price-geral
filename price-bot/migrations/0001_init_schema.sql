-- Raw ingestion table: one row per product the bot saw on a store page,
-- in whichever flow found it. This is intentionally NOT price-back's
-- curated markets/products/market_products shape — the bot doesn't know a
-- product's category, unit, or stable id across stores, and doesn't know a
-- store's coordinates/colors. Turning these raw rows into price-back's
-- schema is a separate curation step (manual or AI-assisted, per the
-- project doc's admin panel), not something the scraper can decide alone.
CREATE TABLE scraped_products (
    id             bigserial PRIMARY KEY,
    store_key      text NOT NULL,              -- StoreConfig.key, e.g. 'carvalhosupershop'
    store_name     text NOT NULL,              -- StoreConfig.name, e.g. 'CarvalhoSuperShop'
    product_name   text NOT NULL,              -- raw scraped name, as-is
    price_text     text NOT NULL,              -- raw scraped price, e.g. 'R$ 7,79'
    price          numeric(10,2),              -- parsed value; NULL if parsing failed
    source         text NOT NULL CHECK (source IN ('extract', 'track')),
    search_term    text,                       -- only set for the 'track' flow
    collected_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_scraped_products_store_collected
    ON scraped_products (store_key, collected_at DESC);

CREATE INDEX idx_scraped_products_product_name
    ON scraped_products (product_name);
