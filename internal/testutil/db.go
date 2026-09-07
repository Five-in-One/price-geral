// Package testutil sets up an isolated connection to the dockerized
// price_user_db_test database for integration tests, migrating it once and
// resetting all tables to their seeded state between tests.
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

	resetToSeed(t, pool)

	t.Cleanup(pool.Close)

	return pool
}

// resetToSeed wipes every data table and re-runs the seed migration so tests
// always see the same known dataset, independent of execution order.
func resetToSeed(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()

	_, err := pool.Exec(ctx, `
		TRUNCATE TABLE
			savings_history,
			monthly_savings,
			economy_summary
		RESTART IDENTITY CASCADE
	`)
	if err != nil {
		t.Fatalf("truncate test tables: %v", err)
	}

	_, err = pool.Exec(ctx, `DELETE FROM schema_migrations WHERE version = '0002_seed_data.sql'`)
	if err != nil {
		t.Fatalf("unmark seed migration: %v", err)
	}

	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("re-seed test database: %v", err)
	}
}
