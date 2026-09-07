package httpserver

import (
	"net/http"
	"sort"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"priceback/internal/geo"
	"priceback/internal/models"
	"priceback/internal/repository"
)

const (
	defaultUserLat  = -5.0892
	defaultUserLng  = -42.8016
	defaultRadiusKm = 5.0
)

func marketsRouter(pool *pgxpool.Pool) chi.Router {
	repo := repository.NewMarketsRepo(pool)

	r := chi.NewRouter()
	r.Get("/", listMarketsHandler(repo))
	return r
}

func listMarketsHandler(repo *repository.MarketsRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		lat, err := floatQueryParam(req, "lat", defaultUserLat)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid lat"})
			return
		}
		lng, err := floatQueryParam(req, "lng", defaultUserLng)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid lng"})
			return
		}
		radiusKm, err := floatQueryParam(req, "radius_km", defaultRadiusKm)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid radius_km"})
			return
		}

		markets, err := repo.ListAll(req.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load markets"})
			return
		}

		results := make([]models.MarketWithDistance, 0, len(markets))
		for _, m := range markets {
			d := geo.HaversineKm(lat, lng, m.Latitude, m.Longitude)
			if d <= radiusKm {
				results = append(results, models.MarketWithDistance{Market: m, DistanceKm: round2(d)})
			}
		}

		sort.Slice(results, func(i, j int) bool {
			return results[i].DistanceKm < results[j].DistanceKm
		})

		writeJSON(w, http.StatusOK, results)
	}
}

func floatQueryParam(r *http.Request, key string, fallback float64) (float64, error) {
	v := r.URL.Query().Get(key)
	if v == "" {
		return fallback, nil
	}
	return strconv.ParseFloat(v, 64)
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}
