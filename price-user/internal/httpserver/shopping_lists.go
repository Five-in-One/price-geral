package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"priceuser/internal/repository"
)

const (
	maxListNameLength = 60
	maxListItems      = 200
	maxListsPerUser   = 50
)

// shoppingListsRouter serves the caller's own saved lists. Mount it behind
// requireAuth: every query is scoped to the token's user id.
func shoppingListsRouter(pool *pgxpool.Pool) chi.Router {
	repo := repository.NewShoppingListsRepo(pool)

	r := chi.NewRouter()
	r.Get("/", listShoppingListsHandler(repo))
	r.Post("/", saveShoppingListHandler(repo))
	r.Delete("/{id}", deleteShoppingListHandler(repo))
	return r
}

func listShoppingListsHandler(repo *repository.ShoppingListsRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		userID, ok := userIDFromContext(req.Context())
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing bearer token"})
			return
		}

		lists, err := repo.List(req.Context(), userID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load shopping lists"})
			return
		}
		writeJSON(w, http.StatusOK, lists)
	}
}

type saveShoppingListRequest struct {
	Name  string            `json:"name"`
	Items []json.RawMessage `json:"items"`
}

// listItem is what the server checks in each item; the rest of it is stored as sent.
type listItem struct {
	Product  map[string]json.RawMessage `json:"product"`
	Quantity int                        `json:"quantity"`
}

// saveShoppingListHandler creates a list, or replaces the one the caller
// already has under the same name (201 vs 200).
func saveShoppingListHandler(repo *repository.ShoppingListsRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		userID, ok := userIDFromContext(req.Context())
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing bearer token"})
			return
		}

		var body saveShoppingListRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, req.Body, 1<<20)).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}

		name := strings.TrimSpace(body.Name)
		if name == "" || utf8.RuneCountInString(name) > maxListNameLength {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name must be 1-60 characters"})
			return
		}
		if len(body.Items) == 0 || len(body.Items) > maxListItems {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "list must have 1-200 items"})
			return
		}
		for _, raw := range body.Items {
			var item listItem
			if err := json.Unmarshal(raw, &item); err != nil || item.Product == nil || item.Quantity < 1 {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "each item needs a product and a quantity of at least 1"})
				return
			}
		}

		items, err := json.Marshal(body.Items)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid items"})
			return
		}

		list, created, err := repo.Save(req.Context(), userID, name, items, maxListsPerUser)
		if err != nil {
			if errors.Is(err, repository.ErrLimitReached) {
				writeJSON(w, http.StatusConflict, map[string]string{"error": "you already have 50 saved lists; delete one first"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save shopping list"})
			return
		}

		status := http.StatusOK
		if created {
			status = http.StatusCreated
		}
		writeJSON(w, status, list)
	}
}

func deleteShoppingListHandler(repo *repository.ShoppingListsRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		userID, ok := userIDFromContext(req.Context())
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing bearer token"})
			return
		}

		if err := repo.Delete(req.Context(), userID, chi.URLParam(req, "id")); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "shopping list not found"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete shopping list"})
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
