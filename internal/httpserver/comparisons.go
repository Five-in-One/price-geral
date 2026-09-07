package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"priceback/internal/models"
	"priceback/internal/repository"
)

func comparisonsRouter(pool *pgxpool.Pool) chi.Router {
	repo := repository.NewComparisonsRepo(pool)

	r := chi.NewRouter()
	r.Post("/", compareHandler(repo))
	return r
}

func compareHandler(repo *repository.ComparisonsRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		var body models.CompareRequest
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}

		for _, item := range body.Items {
			if item.ProductID == "" || item.Quantity <= 0 {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "each item needs a product_id and a positive quantity"})
				return
			}
		}

		results, err := repo.Compare(req.Context(), body.Items)
		if err != nil {
			if errors.Is(err, repository.ErrUnknownProduct) {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown product_id in items"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to compute comparison"})
			return
		}

		writeJSON(w, http.StatusOK, results)
	}
}
