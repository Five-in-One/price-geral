package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"priceback/internal/geo"
	"priceback/internal/repository"
	"priceback/internal/service"
)

// collectHandlers are the controllers of the asynchronous collection flow:
// they validate the request, call service.CollectService (which saves the
// job and publishes the event to the broker) and answer 202 right away.
type collectHandlers struct {
	service  *service.CollectService
	geocoder geo.Geocoder
}

// collectRequest is the Admin panel's "Coletar" form: the same shape the
// bot's synchronous POST /collect accepts.
type collectRequest struct {
	URL          string `json:"url"`
	MarketName   string `json:"market_name"`
	CEP          string `json:"cep"`
	Street       string `json:"street"`
	Number       string `json:"number"`
	Neighborhood string `json:"neighborhood"`
	City         string `json:"city"`
	State        string `json:"state"`
	Complement   string `json:"complement"`
}

// collect handles POST /api/admin/collect.
func (h *collectHandlers) collect(w http.ResponseWriter, req *http.Request) {
	var body collectRequest
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	name := strings.TrimSpace(body.MarketName)
	url := strings.TrimSpace(body.URL)
	if name == "" || url == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "market_name and url are required"})
		return
	}
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "url must start with http:// or https://"})
		return
	}

	market, job, err := h.service.RequestCollect(req.Context(), marketInput(req.Context(), h.geocoder, name, "bot", createMarketRequest{
		CEP:          body.CEP,
		Street:       body.Street,
		Number:       body.Number,
		Neighborhood: body.Neighborhood,
		City:         body.City,
		State:        body.State,
		Complement:   body.Complement,
	}), url)

	var inProgress *service.CollectInProgressError
	switch {
	case errors.As(err, &inProgress):
		writeJSON(w, http.StatusConflict, map[string]any{
			"error":  "collect_in_progress",
			"market": market,
			"job":    inProgress.Job,
		})
	case errors.Is(err, service.ErrBrokerUnavailable):
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "broker_unavailable"})
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to queue collection"})
	default:
		writeJSON(w, http.StatusAccepted, map[string]any{"market": market, "job": job})
	}
}

// reload handles POST /api/markets/{id}/collect ("Recarregar").
func (h *collectHandlers) reload(w http.ResponseWriter, req *http.Request) {
	id := chi.URLParam(req, "id")
	jobs, err := h.service.ReloadMarket(req.Context(), id)
	switch {
	case errors.Is(err, repository.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "market not found"})
	case errors.Is(err, service.ErrNoLinks):
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "market has no links"})
	case errors.Is(err, service.ErrBrokerUnavailable):
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "broker_unavailable"})
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to queue collection"})
	default:
		writeJSON(w, http.StatusAccepted, map[string]any{"market_id": id, "jobs": jobs})
	}
}

// getJob handles GET /api/collect/jobs/{id}, which the panel polls.
func (h *collectHandlers) getJob(w http.ResponseWriter, req *http.Request) {
	job, err := h.service.GetJob(req.Context(), chi.URLParam(req, "id"))
	if err != nil {
		if errors.Is(err, repository.ErrJobNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "job not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load job"})
		return
	}
	writeJSON(w, http.StatusOK, job)
}

// requireBroker answers 503 when the router was built without a broker.
func (h *collectHandlers) requireBroker(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if !h.service.Enabled() {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "broker_unavailable"})
			return
		}
		next(w, req)
	}
}
