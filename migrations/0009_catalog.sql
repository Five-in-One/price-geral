-- Canonical catalog: one row per real-world product, no matter which market
-- sells it. Each market's own listing (a row in `products`) points at the
-- catalog item it is an offer of, which is what lets a shopping list be
-- priced across markets.
--
-- Identity of a catalog item (what makes two listings "the same product"):
--   brand + generic_key + package size (kind, amount, pack count) + sold_by_weight
-- all of them extracted from the scraped name by the bot (migration 0008).
-- The columns are NOT NULL with '' / 0 defaults so the identity can be a plain
-- UNIQUE constraint ("no brand" and "no size" are values, not NULLs).
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE catalog_items (
    id             text PRIMARY KEY,
    name           text NOT NULL,
    category       text NOT NULL DEFAULT 'OUTROS',
    brand          text NOT NULL DEFAULT '',
    generic_key    text NOT NULL,
    size_kind      text NOT NULL DEFAULT '' CHECK (size_kind IN ('', 'mass', 'volume', 'length')),
    size_amount    numeric(14,3) NOT NULL DEFAULT 0,
    pack_count     integer NOT NULL DEFAULT 1 CHECK (pack_count >= 1),
    sold_by_weight boolean NOT NULL DEFAULT false,
    size_label     text NOT NULL DEFAULT '',
    created_at     timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT catalog_items_identity
        UNIQUE (brand, generic_key, size_kind, size_amount, pack_count, sold_by_weight)
);

CREATE INDEX idx_catalog_items_generic_key_trgm ON catalog_items USING gin (generic_key gin_trgm_ops);

-- catalog_locked: a person decided this link in the review queue, so a later
-- re-collection must not silently undo it.
ALTER TABLE products
    ADD COLUMN catalog_id     text REFERENCES catalog_items(id) ON DELETE SET NULL,
    ADD COLUMN catalog_locked boolean NOT NULL DEFAULT false;

CREATE INDEX idx_products_catalog_id ON products (catalog_id);

-- Review queue: "this listing might be the same product as that catalog item".
-- Only doubtful cases land here; obvious ones are linked automatically.
CREATE TABLE catalog_suggestions (
    id          bigserial PRIMARY KEY,
    product_id  text NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    catalog_id  text NOT NULL REFERENCES catalog_items(id) ON DELETE CASCADE,
    score       real NOT NULL,
    reason      text NOT NULL,
    status      text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'accepted', 'rejected')),
    created_at  timestamptz NOT NULL DEFAULT now(),
    decided_at  timestamptz,
    UNIQUE (product_id, catalog_id)
);

CREATE INDEX idx_catalog_suggestions_status ON catalog_suggestions (status);
