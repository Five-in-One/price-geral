-- Owned by price-auth (a separate service), not price-back. Lives here only
-- because both services share this same dockerized Postgres instance.
CREATE DATABASE price_auth_db;
CREATE DATABASE price_auth_db_test;
