package httpserver

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"

	"priceback/internal/geo"
	"priceback/internal/messaging"
	"priceback/internal/repository"
	"priceback/internal/service"
)

// Option customizes the router built by NewRouter.
type Option func(*routerConfig)

type routerConfig struct {
	geocoder  geo.Geocoder
	reverse   geo.ReverseGeocoder
	publisher messaging.Publisher
	guard     authGuard
}

// WithGeocoder turns on address geocoding: markets registered with an
// address (and no explicit coordinates) get real latitude/longitude, which
// is what makes distances to the user meaningful. Without it, markets keep
// the placeholder location and no distance is computed for them.
func WithGeocoder(g geo.Geocoder) Option {
	return func(c *routerConfig) { c.geocoder = g }
}

// WithReverseGeocoder turns on GET /api/geocode/reverse, which the app uses to
// show the address of the device's position. Without it that route answers 503.
func WithReverseGeocoder(g geo.ReverseGeocoder) Option {
	return func(c *routerConfig) { c.reverse = g }
}

// WithPublisher turns on the asynchronous collection routes, which publish
// collect.requested events to the message broker. Without it those routes
// answer 503.
func WithPublisher(p messaging.Publisher) Option {
	return func(c *routerConfig) { c.publisher = p }
}

// WithAuth makes price-back a Resource Server: write and admin routes require
// a valid HS256 token signed by price-auth with this secret (401 without it or
// with a tampered one, 403 with the wrong role). Read routes the app uses
// anonymously (markets, product search, comparisons) stay public.
func WithAuth(jwtSecret string) Option {
	return func(c *routerConfig) { c.guard = authGuard{secret: []byte(jwtSecret)} }
}

// NewRouter builds the full HTTP router for the API, wiring every route
// group against the given database pool.
func NewRouter(pool *pgxpool.Pool, opts ...Option) *chi.Mux {
	var cfg routerConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	r := chi.NewRouter()
	r.Use(instanceHeader)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		// The Flutter app has no login/cookies yet, so a permissive origin
		// list is fine for this dev API (web build, Android/iOS don't need
		// CORS at all).
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		ExposedHeaders:   []string{"X-Instance"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/health", healthHandler(pool))

	collect := &collectHandlers{
		service: service.NewCollectService(
			repository.NewMarketsRepo(pool), repository.NewCollectJobsRepo(pool), cfg.publisher,
		),
		geocoder: cfg.geocoder,
	}

	r.Route("/api", func(api chi.Router) {
		g := cfg.guard
		api.Mount("/markets", marketsRouter(pool, cfg.geocoder, collect, g))
		api.Mount("/products", productsRouter(pool, g))
		api.Mount("/comparisons", comparisonsRouter(pool))
		api.Mount("/catalog", catalogRouter(pool, g))
		api.Mount("/geocode", geocodeRouter(cfg.reverse))
		api.With(g.admin()).Post("/admin/collect", collect.requireBroker(collect.collect))
		api.With(g.admin()).Get("/collect/jobs/{id}", collect.getJob)
	})

	return r
}
