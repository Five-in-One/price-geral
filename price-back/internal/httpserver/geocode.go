package httpserver

import (
	"errors"
	"log"
	"math"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"priceback/internal/geo"
)

func geocodeRouter(reverse geo.ReverseGeocoder) chi.Router {
	r := chi.NewRouter()
	r.Get("/reverse", reverseGeocodeHandler(reverse))
	return r
}

type reverseGeocodeResponse struct {
	Street       string `json:"street"`
	Neighborhood string `json:"neighborhood"`
	City         string `json:"city"`
	State        string `json:"state"`
	CEP          string `json:"cep"`
	// Label is the address as a single line, ready to show.
	Label string `json:"label"`
}

// reverseGeocodeHandler answers "what address is at lat,lng?" — the app calls
// it after reading the device GPS, so the person sees where they are.
func reverseGeocodeHandler(reverse geo.ReverseGeocoder) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		lat, latErr := strconv.ParseFloat(req.URL.Query().Get("lat"), 64)
		lng, lngErr := strconv.ParseFloat(req.URL.Query().Get("lng"), 64)
		if latErr != nil || lngErr != nil || math.IsNaN(lat) || math.IsNaN(lng) ||
			lat < -90 || lat > 90 || lng < -180 || lng > 180 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "lat and lng are required and must be valid coordinates"})
			return
		}
		if reverse == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "reverse geocoding is not configured"})
			return
		}

		place, err := reverse.Reverse(req.Context(), geo.Point{Lat: lat, Lng: lng})
		if err != nil {
			if errors.Is(err, geo.ErrNotFound) {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "no address found for these coordinates"})
				return
			}
			log.Printf("reverse geocode failed: %v", err)
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "address lookup is unavailable right now"})
			return
		}

		writeJSON(w, http.StatusOK, reverseGeocodeResponse{
			Street:       place.Street,
			Neighborhood: place.Neighborhood,
			City:         place.City,
			State:        place.State,
			CEP:          place.CEP,
			Label:        place.Label(),
		})
	}
}
