package main

import (
	"context"
	"log"
	"net/http"

	"github.com/joho/godotenv"

	"priceuser/internal/config"
	"priceuser/internal/db"
	"priceuser/internal/httpserver"
)

func main() {
	// Local dev convenience only — in prod/Docker env vars are injected
	// directly, so a missing .env file here is not an error.
	_ = godotenv.Load()

	cfg := config.Load()

	ctx := context.Background()
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	router := httpserver.NewRouter(pool)

	log.Printf("price-user listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
