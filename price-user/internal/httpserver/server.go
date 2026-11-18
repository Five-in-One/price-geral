package httpserver

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewRouter builds the full HTTP router for price-user, wiring every route
// group against the given database pool. jwtSecret is the secret price-auth
// signs its tokens with; the per-user routes verify against it.
func NewRouter(pool *pgxpool.Pool, jwtSecret string) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/health", healthHandler(pool))

	r.Route("/api", func(api chi.Router) {
		api.Mount("/savings-history", savingsHistoryRouter(pool))
		api.Mount("/economy", economyRouter(pool))

		api.Group(func(protected chi.Router) {
			protected.Use(requireAuth(jwtSecret))
			protected.Mount("/shopping-lists", shoppingListsRouter(pool))
		})
	})

	return r
}
