package httpserver

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"

	"priceauth/internal/models"
	"priceauth/internal/repository"
)

// NewRouter builds the full HTTP router for price-auth, wiring every route
// group against the given database pool and JWT secret.
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

	usersRepo := repository.NewUsersRepo(pool)

	r.Route("/auth", func(auth chi.Router) {
		auth.Post("/register", registerHandler(usersRepo, jwtSecret))
		auth.Post("/login", loginHandler(usersRepo, jwtSecret))

		auth.Group(func(protected chi.Router) {
			protected.Use(requireAuth(jwtSecret))
			protected.Get("/me", meHandler(usersRepo))
		})

		auth.Route("/admin", func(adminOnly chi.Router) {
			adminOnly.Use(requireAuth(jwtSecret))
			adminOnly.Use(requireRole(models.RoleAdmin))
			adminOnly.Post("/users", createUserHandler(usersRepo))
		})
	})

	return r
}
