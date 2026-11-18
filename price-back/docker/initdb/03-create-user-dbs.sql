-- Owned by price-user (a separate service), not price-back. Lives here only
-- because all three services share this same dockerized Postgres instance.
CREATE DATABASE price_user_db;
CREATE DATABASE price_user_db_test;
