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
	r.Post("/list", compareListHandler(repo))
	return r
}

// compareListHandler prices a shopping list of catalog items at every market:
// per-market totals, what each market is missing, the cheapest market for
// each item, and the best combination of markets. See README ("Comparação de
// listas").
func compareListHandler(repo *repository.ComparisonsRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		var body models.ListCompareRequest
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}

		if len(body.Items) == 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "items is required"})
			return
		}
		for _, item := range body.Items {
			if item.CatalogID == "" && item.ProductID == "" {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "each item needs a catalog_id or a product_id"})
				return
			}
			if item.Quantity <= 0 {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "each item needs a positive quantity"})
				return
			}
		}
		if (body.Lat == nil) != (body.Lng == nil) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "lat and lng go together"})
			return
		}
		if body.RadiusKm != nil && (body.Lat == nil || *body.RadiusKm <= 0) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "radius_km needs lat/lng and must be positive"})
			return
		}

		result, err := repo.CompareList(req.Context(), body)
		if err != nil {
			var notInCatalog repository.ProductNotInCatalogError
			switch {
			case errors.As(err, &notInCatalog):
				writeJSON(w, http.StatusBadRequest, map[string]string{
					"error":      "product is not in the catalog yet",
					"product_id": notInCatalog.ProductID,
				})
			case errors.Is(err, repository.ErrUnknownProduct):
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown product_id in items"})
			case errors.Is(err, repository.ErrCatalogItemNotFound):
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown catalog_id in items"})
			default:
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to compute comparison"})
			}
			return
		}

		writeJSON(w, http.StatusOK, result)
	}
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
