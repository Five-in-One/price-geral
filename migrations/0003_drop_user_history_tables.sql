-- savings_history, monthly_savings and economy_summary moved to the
-- price-user service (own database: price_user_db). Drop them here for
-- databases that already ran the old 0001/0002 migrations before the split;
-- a no-op on fresh installs since 0001 no longer creates them.
DROP TABLE IF EXISTS savings_history;
DROP TABLE IF EXISTS monthly_savings;
DROP TABLE IF EXISTS economy_summary;
