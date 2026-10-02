-- A partner company account can be tied to its market in price-back (the
-- market's id there, e.g. "carvalho-super-centro-ab12cd"). Set by the Admin
-- panel when it registers the company together with the market's address,
-- so the Company page knows which market the account's products belong to.
-- NULL for every other account and for companies registered before this.
ALTER TABLE users ADD COLUMN market_id text;
