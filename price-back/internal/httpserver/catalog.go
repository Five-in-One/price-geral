package httpserver

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"priceback/internal/models"
	"priceback/internal/repository"
)

// catalogRouter serves the canonical catalog: real-world products with every
// market's offer, plus the review queue for doubtful matches.
func catalogRouter(pool *pgxpool.Pool, g authGuard) chi.Router {
	repo := repository.NewCatalogRepo(pool)

	r := chi.NewRouter()
	r.Get("/items", listCatalogItemsHandler(repo))
	r.Get("/items/{id}", getCatalogItemHandler(repo))
	r.With(g.admin()).Post("/match", matchCatalogHandler(repo))
	r.With(g.admin()).Get("/suggestions", listSuggestionsHandler(repo))
	r.With(g.admin()).Post("/suggestions/{id}/accept", decideSuggestionHandler(repo, true))
	r.With(g.admin()).Post("/suggestions/{id}/reject", decideSuggestionHandler(repo, false))
	return r
}

func listCatalogItemsHandler(repo *repository.CatalogRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		q := req.URL.Query()

		filter := repository.CatalogFilter{Search: q.Get("search")}
		var err error
		if filter.MinMarkets, err = intQueryParam(req, "min_markets", 0); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid min_markets"})
			return
		}
		if filter.Limit, err = intQueryParam(req, "limit", 0); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid limit"})
			return
		}
		if filter.Offset, err = intQueryParam(req, "offset", 0); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid offset"})
			return
		}

		items, err := repo.ListItems(req.Context(), filter)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load catalog"})
			return
		}
		writeJSON(w, http.StatusOK, items)
	}
}

func getCatalogItemHandler(repo *repository.CatalogRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		item, err := repo.GetItem(req.Context(), chi.URLParam(req, "id"))
		if err != nil {
			if errors.Is(err, repository.ErrCatalogItemNotFound) {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "catalog item not found"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load catalog item"})
			return
		}
		writeJSON(w, http.StatusOK, item)
	}
}

// matchCatalogHandler links every listing that has no catalog item yet.
// With ?all=true it re-evaluates every listing a person hasn't locked.
func matchCatalogHandler(repo *repository.CatalogRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		summary, err := repo.MatchUnlinked(req.Context(), req.URL.Query().Get("all") == "true")
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to match products"})
			return
		}
		writeJSON(w, http.StatusOK, summary)
	}
}

func listSuggestionsHandler(repo *repository.CatalogRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		status := req.URL.Query().Get("status")
		if status != "" && status != "pending" && status != "accepted" && status != "rejected" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "status must be pending, accepted or rejected"})
			return
		}
		limit, err := intQueryParam(req, "limit", 0)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid limit"})
			return
		}

		suggestions, err := repo.ListSuggestions(req.Context(), status, limit)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load suggestions"})
			return
		}
		if suggestions == nil {
			suggestions = []models.CatalogSuggestion{}
		}
		writeJSON(w, http.StatusOK, suggestions)
	}
}

func decideSuggestionHandler(repo *repository.CatalogRepo, accept bool) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(req, "id"), 10, 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid suggestion id"})
			return
		}

		if accept {
			err = repo.AcceptSuggestion(req.Context(), id)
		} else {
			err = repo.RejectSuggestion(req.Context(), id)
		}
		switch {
		case err == nil:
			w.WriteHeader(http.StatusNoContent)
		case errors.Is(err, repository.ErrSuggestionNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "suggestion not found"})
		case errors.Is(err, repository.ErrSuggestionDecided):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "suggestion already decided"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save decision"})
		}
	}
}

func intQueryParam(r *http.Request, key string, fallback int) (int, error) {
	v := r.URL.Query().Get(key)
	if v == "" {
		return fallback, nil
	}
	return strconv.Atoi(v)
}
