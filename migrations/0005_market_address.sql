-- Lets the Admin panel register a market's storefront link and physical
-- address up front, in "coleta sob demanda", before any product is
-- scraped — so bot-collected markets stop landing on the same fixed
-- default coordinates. All columns are nullable: markets registered
-- before this migration (seed/company) simply have them empty.
ALTER TABLE markets ADD COLUMN site_url     text;
ALTER TABLE markets ADD COLUMN cep          text;
ALTER TABLE markets ADD COLUMN street       text;
ALTER TABLE markets ADD COLUMN number       text;
ALTER TABLE markets ADD COLUMN neighborhood text;
ALTER TABLE markets ADD COLUMN city         text;
ALTER TABLE markets ADD COLUMN state        text;
ALTER TABLE markets ADD COLUMN complement   text;
