-- A market can have more than one storefront link (different domains for
-- the same chain, a site that moved, etc.), and matching which market a
-- collection belongs to is still done by name, not by URL — so links live
-- in their own table instead of the single `site_url` column added in
-- 0005. The Admin panel's "Recarregar" button re-runs the bot's collection
-- against every link stored here for a market.
CREATE TABLE market_links (
    id          text PRIMARY KEY,
    market_id   text NOT NULL REFERENCES markets(id) ON DELETE CASCADE,
    url         text NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_market_links_market_url ON market_links (market_id, lower(url));

INSERT INTO market_links (id, market_id, url)
SELECT id || '-link-1', id, site_url FROM markets WHERE site_url IS NOT NULL AND site_url <> '';

ALTER TABLE markets DROP COLUMN site_url;
