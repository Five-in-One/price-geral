package main

import (
	"context"
	"log"
	"net/http"

	"github.com/joho/godotenv"

	"priceback/internal/config"
	"priceback/internal/db"
	"priceback/internal/geo"
	"priceback/internal/httpserver"
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

	// One client for both directions, so they share Nominatim's rate limit.
	nominatim := geo.NewNominatim("PriceApp/1.0 (projeto-faculdade; price-back)")
	router := httpserver.NewRouter(pool,
		httpserver.WithGeocoder(nominatim),
		httpserver.WithReverseGeocoder(nominatim),
	)

	log.Printf("price-back listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
