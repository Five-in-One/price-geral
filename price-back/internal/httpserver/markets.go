package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

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

func marketsRouter(pool *pgxpool.Pool, geocoder geo.Geocoder, collect *collectHandlers, g authGuard) chi.Router {
	repo := repository.NewMarketsRepo(pool)

	r := chi.NewRouter()
	r.Get("/", listMarketsHandler(repo))
	r.With(g.admin()).Post("/", createMarketHandler(repo, geocoder))
	r.With(g.staff()).Patch("/{id}", updateMarketHandler(repo, geocoder))
	r.With(g.admin()).Delete("/{id}", deleteMarketHandler(repo))
	r.With(g.admin()).Post("/{id}/links", addMarketLinkHandler(repo))
	r.With(g.admin()).Delete("/{id}/links", removeMarketLinkHandler(repo))
	r.With(g.admin()).Post("/{id}/collect", collect.requireBroker(collect.reload))
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

			// A market still on its placeholder location has no meaningful
			// distance: keep it in the list, without a distance, rather than
			// hide it or invent a number.
			if !m.HasLocation {
				results = append(results, models.MarketWithDistance{Market: m})
				continue
			}

			d := geo.HaversineKm(lat, lng, m.Latitude, m.Longitude)
			if showAll || d <= radiusKm {
				km := round2(d)
				results = append(results, models.MarketWithDistance{Market: m, DistanceKm: &km})
			}
		}

		if showAll {
			sort.Slice(results, func(i, j int) bool {
				return results[i].Name < results[j].Name
			})
		} else {
			// Nearest first; markets with no known distance go last.
			sort.SliceStable(results, func(i, j int) bool {
				a, b := results[i].DistanceKm, results[j].DistanceKm
				switch {
				case a == nil:
					return false
				case b == nil:
					return true
				default:
					return *a < *b
				}
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

func createMarketHandler(repo *repository.MarketsRepo, geocoder geo.Geocoder) http.HandlerFunc {
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

		market, err := repo.FindOrCreate(req.Context(), marketInput(req.Context(), geocoder, name, source, body))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create market"})
			return
		}

		writeJSON(w, http.StatusCreated, market)
	}
}

// marketInput turns a market registration body into the repository input:
// coordinates come from the body or from geocoding the address, and
// initials and colors are derived from the name when not sent.
func marketInput(ctx context.Context, geocoder geo.Geocoder, name, source string, body createMarketRequest) repository.CreateMarketInput {
	lat, lng := defaultUserLat, defaultUserLng
	hasLocation := false
	if body.Latitude != nil && body.Longitude != nil {
		lat, lng, hasLocation = *body.Latitude, *body.Longitude, true
	} else if pt, ok := geocodeAddress(ctx, geocoder, name, geo.Address{
		CEP:          strings.TrimSpace(body.CEP),
		Street:       strings.TrimSpace(body.Street),
		Number:       strings.TrimSpace(body.Number),
		Neighborhood: strings.TrimSpace(body.Neighborhood),
		City:         strings.TrimSpace(body.City),
		State:        strings.TrimSpace(body.State),
	}); ok {
		// Coordinates weren't sent: derived from the address so the
		// distance shown to users is real. A failed lookup is not fatal —
		// the market is still registered, just without a location.
		lat, lng, hasLocation = pt.Lat, pt.Lng, true
	}

	initials := body.Initials
	if initials == "" {
		initials = deriveInitials(name)
	}

	primary, background := body.PrimaryColor, body.BackgroundColor
	if primary == "" || background == "" {
		primary, background = pickPalette(name)
	}

	return repository.CreateMarketInput{
		Name:            name,
		Initials:        initials,
		Latitude:        lat,
		Longitude:       lng,
		HasLocation:     hasLocation,
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
	}
}

// geocodeAddress resolves addr to coordinates, or reports ok=false when
// there is no geocoder, nothing to look up, or the lookup failed (logged).
func geocodeAddress(ctx context.Context, geocoder geo.Geocoder, marketName string, addr geo.Address) (geo.Point, bool) {
	if geocoder == nil || addr.Empty() {
		return geo.Point{}, false
	}
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	pt, err := geocoder.Geocode(ctx, addr)
	if err != nil {
		log.Printf("geocoding market %q failed: %v", marketName, err)
		return geo.Point{}, false
	}
	return pt, true
}

type updateMarketRequest struct {
	Name            *string `json:"name"`
	Initials        *string `json:"initials"`
	PrimaryColor    *string `json:"primary_color"`
	BackgroundColor *string `json:"background_color"`
	CEP             *string `json:"cep"`
	Street          *string `json:"street"`
	Number          *string `json:"number"`
	Neighborhood    *string `json:"neighborhood"`
	City            *string `json:"city"`
	State           *string `json:"state"`
	Complement      *string `json:"complement"`
}

// trimmed returns s with surrounding spaces removed (nil stays nil).
func trimmed(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	return &t
}

// orCurrent returns *in when provided, otherwise cur.
func orCurrent(in *string, cur string) string {
	if in != nil {
		return *in
	}
	return cur
}

func updateMarketHandler(repo *repository.MarketsRepo, geocoder geo.Geocoder) http.HandlerFunc {
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

		input := repository.UpdateMarketInput{
			Name:            body.Name,
			Initials:        body.Initials,
			PrimaryColor:    body.PrimaryColor,
			BackgroundColor: body.BackgroundColor,
			CEP:             trimmed(body.CEP),
			Street:          trimmed(body.Street),
			Number:          trimmed(body.Number),
			Neighborhood:    trimmed(body.Neighborhood),
			City:            trimmed(body.City),
			State:           trimmed(body.State),
			Complement:      trimmed(body.Complement),
		}

		// When the address changes, the stored coordinates describe the old
		// one: look the new one up, and if that fails drop the location
		// rather than keep a distance that is now wrong.
		current, err := repo.GetByID(req.Context(), id)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "market not found"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update market"})
			return
		}
		addr := geo.Address{
			CEP:          orCurrent(input.CEP, current.CEP),
			Street:       orCurrent(input.Street, current.Street),
			Number:       orCurrent(input.Number, current.Number),
			Neighborhood: orCurrent(input.Neighborhood, current.Neighborhood),
			City:         orCurrent(input.City, current.City),
			State:        orCurrent(input.State, current.State),
		}
		before := geo.Address{
			CEP: current.CEP, Street: current.Street, Number: current.Number,
			Neighborhood: current.Neighborhood, City: current.City, State: current.State,
		}
		if addr != before {
			name := orCurrent(input.Name, current.Name)
			if pt, ok := geocodeAddress(req.Context(), geocoder, name, addr); ok {
				input.Location = &repository.LocationUpdate{Latitude: pt.Lat, Longitude: pt.Lng, HasLocation: true}
			} else {
				input.Location = &repository.LocationUpdate{Latitude: current.Latitude, Longitude: current.Longitude, HasLocation: false}
			}
		}

		market, err := repo.Update(req.Context(), id, input)
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
