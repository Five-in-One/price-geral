-- Structured data the bot extracts from a scraped product name (brand, package
-- size, a brand-free "generic key"), so the same item can be recognised across
-- markets and compared by unit price. All optional: products registered through
-- the partner page, or before this migration, simply have them empty.
--
--   size_kind / size_amount   whole pack, in a base unit: g (mass), ml (volume) or
--                             m (length). Both set or both NULL.
--   pack_count                pieces in the pack (12 rolls, 30 bars, ...), default 1
--   sold_by_weight            "Queijo ... KG": the listed price is per kg
--   generic_key               what the product is without brand/size/packaging,
--                             e.g. "arroz branco tipo1" for Camil and Sofia alike
ALTER TABLE products
    ADD COLUMN brand          text,
    ADD COLUMN generic_key    text,
    ADD COLUMN pack_count     integer NOT NULL DEFAULT 1 CHECK (pack_count >= 1),
    ADD COLUMN size_kind      text CHECK (size_kind IN ('mass', 'volume', 'length')),
    ADD COLUMN size_amount    numeric(14,3) CHECK (size_amount > 0),
    ADD COLUMN size_label     text,
    ADD COLUMN sold_by_weight boolean NOT NULL DEFAULT false,
    ADD CONSTRAINT products_size_pair CHECK ((size_kind IS NULL) = (size_amount IS NULL));

CREATE INDEX idx_products_generic_key ON products (generic_key) WHERE generic_key IS NOT NULL;

-- When each price was observed. Rows that already exist get the migration time:
-- the real collection time was never sent to price-back.
ALTER TABLE market_products ADD COLUMN collected_at timestamptz NOT NULL DEFAULT now();
