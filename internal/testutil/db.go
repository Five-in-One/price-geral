// Package testutil sets up an isolated connection to the dockerized
// price_user_db_test database for integration tests, migrating it once and
// resetting all tables to a known fixture dataset between tests.
package testutil

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"priceuser/internal/db"
)

const defaultTestDatabaseURL = "postgres://postgres:postgres@localhost:5432/price_user_db_test?sslmode=disable"

// NewTestPool connects to the test database, ensures migrations are applied,
// resets all tables to their seeded state, and registers cleanup to close
// the pool when the test finishes.
func NewTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		url = defaultTestDatabaseURL
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("ping test database (is `docker compose up -d` running in price-back?): %v", err)
	}

	if err := db.Migrate(ctx, pool); err != nil {
		pool.Close()
		t.Fatalf("migrate test database: %v", err)
	}

	resetToFixture(t, pool)

	t.Cleanup(pool.Close)

	return pool
}

// fixtureSQL is a test-only dataset (production no longer seeds demo data;
// see migrations/0002_seed_data.sql) that this package's tests assert against.
const fixtureSQL = `
INSERT INTO savings_history (date_label, market_name, items_count, total_cost, savings, occurred_at) VALUES
    ('24 MAI', 'Atacado Família', 12, 187.40, 28.60, '2026-05-24'),
    ('17 MAI', 'Super Economia',  15, 215.10, 18.50, '2026-05-17'),
    ('10 MAI', 'Atacado Família', 8,  112.30, 14.20, '2026-05-10');

INSERT INTO monthly_savings (label, value, sort_order, is_active) VALUES
    ('Dez', 71,  1, false),
    ('Jan', 84,  2, false),
    ('Fev', 65,  3, false),
    ('Mar', 96,  4, false),
    ('Abr', 89,  5, false),
    ('Mai', 112, 6, true);

INSERT INTO economy_summary (total_saved, month_label, month_spend, month_savings, month_savings_percent) VALUES
    (517.10, 'Maio de 2026', 534.60, 112.30, 17.00);
`

// resetToFixture wipes every data table and inserts fixtureSQL so tests
// always see the same known dataset, independent of execution order.
func resetToFixture(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()

	_, err := pool.Exec(ctx, `
		TRUNCATE TABLE
			savings_history,
			monthly_savings,
			economy_summary,
			shopping_lists
		RESTART IDENTITY CASCADE
	`)
	if err != nil {
		t.Fatalf("truncate test tables: %v", err)
	}

	if _, err := pool.Exec(ctx, fixtureSQL); err != nil {
		t.Fatalf("insert test fixture: %v", err)
	}
}
