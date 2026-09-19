package httpserver_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"

	"priceuser/internal/httpserver"
	"priceuser/internal/models"
	"priceuser/internal/testutil"
)

const testSecret = "test-secret"

func tokenFor(t *testing.T, userID string) string {
	t.Helper()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  userID,
		"username": "someone",
		"role":     "user",
		"exp":      time.Now().Add(time.Hour).Unix(),
	}).SignedString([]byte(testSecret))
	require.NoError(t, err)
	return token
}

func doListRequest(t *testing.T, router http.Handler, method, path string, body any, token string) *httptest.ResponseRecorder {
	t.Helper()
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		require.NoError(t, err)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func sampleItems() []map[string]any {
	return []map[string]any{
		{"product": map[string]any{"id": "p1", "name": "Arroz 5kg", "catalog_id": "c1"}, "quantity": 2},
		{"product": map[string]any{"id": "p2", "name": "Feijão 1kg", "catalog_id": "c2"}, "quantity": 1},
	}
}

func TestShoppingLists_RequireAuth(t *testing.T) {
	router := httpserver.NewRouter(testutil.NewTestPool(t), testSecret)

	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/shopping-lists"},
		{http.MethodPost, "/api/shopping-lists"},
		{http.MethodDelete, "/api/shopping-lists/abc"},
	} {
		rec := doListRequest(t, router, tc.method, tc.path, nil, "")
		require.Equal(t, http.StatusUnauthorized, rec.Code, tc.method+" "+tc.path)
	}

	rec := doListRequest(t, router, http.MethodGet, "/api/shopping-lists", nil, "not-a-token")
	require.Equal(t, http.StatusUnauthorized, rec.Code)

	// A token signed with another secret is rejected too.
	other, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"user_id": "u1"}).SignedString([]byte("other-secret"))
	require.NoError(t, err)
	rec = doListRequest(t, router, http.MethodGet, "/api/shopping-lists", nil, other)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestShoppingLists_SaveListAndDelete(t *testing.T) {
	router := httpserver.NewRouter(testutil.NewTestPool(t), testSecret)
	token := tokenFor(t, "user-1")

	empty := doListRequest(t, router, http.MethodGet, "/api/shopping-lists", nil, token)
	require.Equal(t, http.StatusOK, empty.Code)
	require.JSONEq(t, "[]", empty.Body.String())

	created := doListRequest(t, router, http.MethodPost, "/api/shopping-lists",
		map[string]any{"name": "  Compras do mês ", "items": sampleItems()}, token)
	require.Equal(t, http.StatusCreated, created.Code)

	var list models.ShoppingList
	require.NoError(t, json.Unmarshal(created.Body.Bytes(), &list))
	require.NotEmpty(t, list.ID)
	require.Equal(t, "Compras do mês", list.Name)
	require.Equal(t, 2, list.ItemCount)

	// The items come back exactly as sent.
	var items []map[string]any
	require.NoError(t, json.Unmarshal(list.Items, &items))
	require.Equal(t, "Arroz 5kg", items[0]["product"].(map[string]any)["name"])
	require.EqualValues(t, 2, items[0]["quantity"])

	listed := doListRequest(t, router, http.MethodGet, "/api/shopping-lists", nil, token)
	var lists []models.ShoppingList
	require.NoError(t, json.Unmarshal(listed.Body.Bytes(), &lists))
	require.Len(t, lists, 1)
	require.Equal(t, list.ID, lists[0].ID)

	deleted := doListRequest(t, router, http.MethodDelete, "/api/shopping-lists/"+list.ID, nil, token)
	require.Equal(t, http.StatusNoContent, deleted.Code)
	again := doListRequest(t, router, http.MethodDelete, "/api/shopping-lists/"+list.ID, nil, token)
	require.Equal(t, http.StatusNotFound, again.Code)
}

func TestShoppingLists_SameNameReplacesTheList(t *testing.T) {
	router := httpserver.NewRouter(testutil.NewTestPool(t), testSecret)
	token := tokenFor(t, "user-1")

	first := doListRequest(t, router, http.MethodPost, "/api/shopping-lists",
		map[string]any{"name": "Semana", "items": sampleItems()}, token)
	require.Equal(t, http.StatusCreated, first.Code)
	var created models.ShoppingList
	require.NoError(t, json.Unmarshal(first.Body.Bytes(), &created))

	second := doListRequest(t, router, http.MethodPost, "/api/shopping-lists",
		map[string]any{"name": "SEMANA", "items": sampleItems()[:1]}, token)
	require.Equal(t, http.StatusOK, second.Code)
	var updated models.ShoppingList
	require.NoError(t, json.Unmarshal(second.Body.Bytes(), &updated))
	require.Equal(t, created.ID, updated.ID)
	require.Equal(t, 1, updated.ItemCount)

	listed := doListRequest(t, router, http.MethodGet, "/api/shopping-lists", nil, token)
	var lists []models.ShoppingList
	require.NoError(t, json.Unmarshal(listed.Body.Bytes(), &lists))
	require.Len(t, lists, 1)
}

func TestShoppingLists_AreScopedToTheirOwner(t *testing.T) {
	router := httpserver.NewRouter(testutil.NewTestPool(t), testSecret)
	alice, bob := tokenFor(t, "alice"), tokenFor(t, "bob")

	created := doListRequest(t, router, http.MethodPost, "/api/shopping-lists",
		map[string]any{"name": "Só da Alice", "items": sampleItems()}, alice)
	require.Equal(t, http.StatusCreated, created.Code)
	var list models.ShoppingList
	require.NoError(t, json.Unmarshal(created.Body.Bytes(), &list))

	bobLists := doListRequest(t, router, http.MethodGet, "/api/shopping-lists", nil, bob)
	require.JSONEq(t, "[]", bobLists.Body.String())

	// Bob can't delete Alice's list, and her list is still there afterwards.
	stolen := doListRequest(t, router, http.MethodDelete, "/api/shopping-lists/"+list.ID, nil, bob)
	require.Equal(t, http.StatusNotFound, stolen.Code)
	aliceLists := doListRequest(t, router, http.MethodGet, "/api/shopping-lists", nil, alice)
	var lists []models.ShoppingList
	require.NoError(t, json.Unmarshal(aliceLists.Body.Bytes(), &lists))
	require.Len(t, lists, 1)

	// Bob can use the same name for his own list.
	bobSaved := doListRequest(t, router, http.MethodPost, "/api/shopping-lists",
		map[string]any{"name": "Só da Alice", "items": sampleItems()}, bob)
	require.Equal(t, http.StatusCreated, bobSaved.Code)
}

func TestShoppingLists_RejectInvalidInput(t *testing.T) {
	router := httpserver.NewRouter(testutil.NewTestPool(t), testSecret)
	token := tokenFor(t, "user-1")

	tooManyItems := make([]map[string]any, 201)
	for i := range tooManyItems {
		tooManyItems[i] = sampleItems()[0]
	}

	for name, body := range map[string]map[string]any{
		"blank name":      {"name": "   ", "items": sampleItems()},
		"long name":       {"name": strings.Repeat("a", 61), "items": sampleItems()},
		"no items":        {"name": "Vazia", "items": []any{}},
		"too many items":  {"name": "Enorme", "items": tooManyItems},
		"zero quantity":   {"name": "Zero", "items": []map[string]any{{"product": map[string]any{"id": "p1"}, "quantity": 0}}},
		"missing product": {"name": "Sem produto", "items": []map[string]any{{"quantity": 1}}},
	} {
		rec := doListRequest(t, router, http.MethodPost, "/api/shopping-lists", body, token)
		require.Equal(t, http.StatusBadRequest, rec.Code, name)
	}

	rec := doListRequest(t, router, http.MethodDelete, "/api/shopping-lists/not-a-uuid", nil, token)
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestShoppingLists_LimitPerUser(t *testing.T) {
	router := httpserver.NewRouter(testutil.NewTestPool(t), testSecret)
	token := tokenFor(t, "user-1")

	for i := 0; i < 50; i++ {
		rec := doListRequest(t, router, http.MethodPost, "/api/shopping-lists",
			map[string]any{"name": "Lista " + strings.Repeat("x", i), "items": sampleItems()}, token)
		require.Equal(t, http.StatusCreated, rec.Code)
	}

	over := doListRequest(t, router, http.MethodPost, "/api/shopping-lists",
		map[string]any{"name": "Uma a mais", "items": sampleItems()}, token)
	require.Equal(t, http.StatusConflict, over.Code)

	// Replacing an existing list is still allowed at the limit.
	replace := doListRequest(t, router, http.MethodPost, "/api/shopping-lists",
		map[string]any{"name": "Lista ", "items": sampleItems()[:1]}, token)
	require.Equal(t, http.StatusOK, replace.Code)
}
