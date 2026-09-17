package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"

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
	r.Post("/", createMarketHandler(repo))
	r.Patch("/{id}", updateMarketHandler(repo))
	r.Delete("/{id}", deleteMarketHandler(repo))
	r.Post("/{id}/links", addMarketLinkHandler(repo))
	r.Delete("/{id}/links", removeMarketLinkHandler(repo))
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
		// Admin panel view: ignore distance entirely and optionally filter
		// by name, so it can browse every registered market.
		showAll := req.URL.Query().Get("all") == "true"
		search := strings.ToLower(strings.TrimSpace(req.URL.Query().Get("search")))

		markets, err := repo.ListAll(req.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load markets"})
			return
		}

		results := make([]models.MarketWithDistance, 0, len(markets))
		for _, m := range markets {
			if search != "" && !strings.Contains(strings.ToLower(m.Name), search) {
				continue
			}

			d := geo.HaversineKm(lat, lng, m.Latitude, m.Longitude)
			if showAll || d <= radiusKm {
				results = append(results, models.MarketWithDistance{Market: m, DistanceKm: round2(d)})
			}
		}

		if showAll {
			sort.Slice(results, func(i, j int) bool {
				return results[i].Name < results[j].Name
			})
		} else {
			sort.Slice(results, func(i, j int) bool {
				return results[i].DistanceKm < results[j].DistanceKm
			})
		}

		writeJSON(w, http.StatusOK, results)
	}
}

type createMarketRequest struct {
	Name            string   `json:"name"`
	Initials        string   `json:"initials"`
	Latitude        *float64 `json:"latitude"`
	Longitude       *float64 `json:"longitude"`
	PrimaryColor    string   `json:"primary_color"`
	BackgroundColor string   `json:"background_color"`
	Source          string   `json:"source"`
	Link            string   `json:"link"`
	CEP             string   `json:"cep"`
	Street          string   `json:"street"`
	Number          string   `json:"number"`
	Neighborhood    string   `json:"neighborhood"`
	City            string   `json:"city"`
	State           string   `json:"state"`
	Complement      string   `json:"complement"`
}

func createMarketHandler(repo *repository.MarketsRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		var body createMarketRequest
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}

		name := strings.TrimSpace(body.Name)
		if name == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
			return
		}

		source := body.Source
		if source == "" {
			source = "company"
		}
		if source != "company" && source != "bot" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "source must be 'company' or 'bot'"})
			return
		}

		lat, lng := defaultUserLat, defaultUserLng
		if body.Latitude != nil {
			lat = *body.Latitude
		}
		if body.Longitude != nil {
			lng = *body.Longitude
		}

		initials := body.Initials
		if initials == "" {
			initials = deriveInitials(name)
		}

		primary, background := body.PrimaryColor, body.BackgroundColor
		if primary == "" || background == "" {
			primary, background = pickPalette(name)
		}

		market, err := repo.FindOrCreate(req.Context(), repository.CreateMarketInput{
			Name:            name,
			Initials:        initials,
			Latitude:        lat,
			Longitude:       lng,
			PrimaryColor:    primary,
			BackgroundColor: background,
			Source:          source,
			Link:            strings.TrimSpace(body.Link),
			CEP:             strings.TrimSpace(body.CEP),
			Street:          strings.TrimSpace(body.Street),
			Number:          strings.TrimSpace(body.Number),
			Neighborhood:    strings.TrimSpace(body.Neighborhood),
			City:            strings.TrimSpace(body.City),
			State:           strings.TrimSpace(body.State),
			Complement:      strings.TrimSpace(body.Complement),
		})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create market"})
			return
		}

		writeJSON(w, http.StatusCreated, market)
	}
}

type updateMarketRequest struct {
	Name            *string `json:"name"`
	Initials        *string `json:"initials"`
	PrimaryColor    *string `json:"primary_color"`
	BackgroundColor *string `json:"background_color"`
}

func updateMarketHandler(repo *repository.MarketsRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		id := chi.URLParam(req, "id")

		var body updateMarketRequest
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}

		if body.Name != nil {
			trimmed := strings.TrimSpace(*body.Name)
			if trimmed == "" {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name cannot be empty"})
				return
			}
			body.Name = &trimmed
		}

		market, err := repo.Update(req.Context(), id, repository.UpdateMarketInput{
			Name:            body.Name,
			Initials:        body.Initials,
			PrimaryColor:    body.PrimaryColor,
			BackgroundColor: body.BackgroundColor,
		})
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "market not found"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update market"})
			return
		}

		writeJSON(w, http.StatusOK, market)
	}
}

func deleteMarketHandler(repo *repository.MarketsRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		id := chi.URLParam(req, "id")

		if err := repo.Delete(req.Context(), id); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "market not found"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete market"})
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

type addMarketLinkRequest struct {
	URL string `json:"url"`
}

func addMarketLinkHandler(repo *repository.MarketsRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		id := chi.URLParam(req, "id")

		var body addMarketLinkRequest
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}

		url := strings.TrimSpace(body.URL)
		if url == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "url is required"})
			return
		}

		market, err := repo.AddLink(req.Context(), id, url)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "market not found"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to add link"})
			return
		}

		writeJSON(w, http.StatusOK, market)
	}
}

func removeMarketLinkHandler(repo *repository.MarketsRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		id := chi.URLParam(req, "id")
		url := strings.TrimSpace(req.URL.Query().Get("url"))
		if url == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "url query param is required"})
			return
		}

		market, err := repo.RemoveLink(req.Context(), id, url)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "market not found"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to remove link"})
			return
		}

		writeJSON(w, http.StatusOK, market)
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
