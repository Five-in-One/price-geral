-- Admin panel's "coleta sob demanda" is a third origin for raw scraped
-- rows, distinct from the extract/track CLI flows.
ALTER TABLE scraped_products DROP CONSTRAINT scraped_products_source_check;
ALTER TABLE scraped_products ADD CONSTRAINT scraped_products_source_check
    CHECK (source IN ('extract', 'track', 'admin_link'));
