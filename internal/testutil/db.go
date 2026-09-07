// Package testutil sets up an isolated connection to the dockerized
// price_auth_db_test database for integration tests, migrating it once and
// truncating all tables between tests so each test starts from a clean
// slate.
package testutil

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"priceauth/internal/db"
)

const defaultTestDatabaseURL = "postgres://postgres:postgres@localhost:5432/price_auth_db_test?sslmode=disable"

// NewTestPool connects to the test database, ensures migrations are
// applied, truncates every table, and registers cleanup to close the pool
// when the test finishes.
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

	if _, err := pool.Exec(ctx, `TRUNCATE TABLE users RESTART IDENTITY CASCADE`); err != nil {
		pool.Close()
		t.Fatalf("truncate test tables: %v", err)
	}

	// The seed migration only runs once; re-run it after truncating so
	// every test still has the bootstrap admin@price.com account.
	if _, err := pool.Exec(ctx, `DELETE FROM schema_migrations WHERE version = '0003_seed_admin.sql'`); err != nil {
		pool.Close()
		t.Fatalf("unmark seed migration: %v", err)
	}
	if err := db.Migrate(ctx, pool); err != nil {
		pool.Close()
		t.Fatalf("re-seed test database: %v", err)
	}

	t.Cleanup(pool.Close)

	return pool
}
