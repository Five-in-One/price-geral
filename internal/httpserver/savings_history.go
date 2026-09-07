package httpserver

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"priceuser/internal/models"
	"priceuser/internal/repository"
)

func savingsHistoryRouter(pool *pgxpool.Pool) chi.Router {
	repo := repository.NewSavingsHistoryRepo(pool)

	r := chi.NewRouter()
	r.Get("/", listSavingsHistoryHandler(repo))
	return r
}

func listSavingsHistoryHandler(repo *repository.SavingsHistoryRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		items, err := repo.List(req.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load savings history"})
			return
		}
		if items == nil {
			items = []models.SavingsHistoryItem{}
		}
		writeJSON(w, http.StatusOK, items)
	}
}
