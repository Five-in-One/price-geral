package httpserver

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"

	"priceback/internal/geo"
)

// Option customizes the router built by NewRouter.
type Option func(*routerConfig)

type routerConfig struct {
	geocoder geo.Geocoder
}

// WithGeocoder turns on address geocoding: markets registered with an
// address (and no explicit coordinates) get real latitude/longitude, which
// is what makes distances to the user meaningful. Without it, markets keep
// the placeholder location and no distance is computed for them.
func WithGeocoder(g geo.Geocoder) Option {
	return func(c *routerConfig) { c.geocoder = g }
}

// NewRouter builds the full HTTP router for the API, wiring every route
// group against the given database pool.
func NewRouter(pool *pgxpool.Pool, opts ...Option) *chi.Mux {
	var cfg routerConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		// The Flutter app has no login/cookies yet, so a permissive origin
		// list is fine for this dev API (web build, Android/iOS don't need
		// CORS at all).
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/health", healthHandler(pool))

	r.Route("/api", func(api chi.Router) {
		api.Mount("/markets", marketsRouter(pool, cfg.geocoder))
		api.Mount("/products", productsRouter(pool))
		api.Mount("/comparisons", comparisonsRouter(pool))
		api.Mount("/catalog", catalogRouter(pool))
	})

	return r
}
