package httpserver

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"priceuser/internal/repository"
)

func economyRouter(pool *pgxpool.Pool) chi.Router {
	repo := repository.NewEconomyRepo(pool)

	r := chi.NewRouter()
	r.Get("/summary", economySummaryHandler(repo))
	return r
}

func economySummaryHandler(repo *repository.EconomyRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		summary, err := repo.Summary(req.Context())
		if err != nil {
			if errors.Is(err, repository.ErrNoEconomySummary) {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "no economy summary configured"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load economy summary"})
			return
		}
		writeJSON(w, http.StatusOK, summary)
	}
}
