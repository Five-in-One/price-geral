package httpserver_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"priceback/internal/httpserver"
	"priceback/internal/models"
	"priceback/internal/testutil"
)

func TestListMarkets_DefaultRadius_ReturnsAllSortedByDistance(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	req := httptest.NewRequest(http.MethodGet, "/api/markets", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var results []models.MarketWithDistance
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &results))
	require.Len(t, results, 5)

	wantOrder := []string{"md", "se", "hc", "af", "ms"}
	gotOrder := make([]string, len(results))
	for i, m := range results {
		gotOrder[i] = m.ID
	}
	require.Equal(t, wantOrder, gotOrder)

	require.InDelta(t, 0.4, results[0].DistanceKm, 0.05)
	require.Equal(t, "Mercado do Bairro", results[0].Name)
	require.Equal(t, "MD", results[0].Initials)
	require.Equal(t, "#1F9E5F", results[0].PrimaryColor)
}

func TestListMarkets_SmallRadius_FiltersFartherMarkets(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	req := httptest.NewRequest(http.MethodGet, "/api/markets?radius_km=1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var results []models.MarketWithDistance
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &results))
	require.Len(t, results, 2)
	require.Equal(t, "md", results[0].ID)
	require.Equal(t, "se", results[1].ID)
}

func TestListMarkets_InvalidLat_ReturnsBadRequest(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	req := httptest.NewRequest(http.MethodGet, "/api/markets?lat=not-a-number", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestListMarkets_SeedData_HasSourceSeed(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	req := httptest.NewRequest(http.MethodGet, "/api/markets", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var results []models.MarketWithDistance
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &results))
	for _, m := range results {
		require.Equal(t, "seed", m.Source)
	}
}

func TestListMarkets_All_IgnoresRadiusAndFiltersBySearch(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	// radius_km=0 would normally exclude every seeded market.
	req := httptest.NewRequest(http.MethodGet, "/api/markets?all=true&radius_km=0&search=hortifruti", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var results []models.MarketWithDistance
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &results))
	require.Len(t, results, 1)
	require.Equal(t, "hc", results[0].ID)
}

func postJSON(t *testing.T, router http.Handler, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestCreateMarket_MinimalBody_FillsDefaults(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	rec := postJSON(t, router, "/api/markets", map[string]string{"name": "Mercadão Popular"})
	require.Equal(t, http.StatusCreated, rec.Code)

	var market models.Market
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &market))
	require.Equal(t, "Mercadão Popular", market.Name)
	require.Equal(t, "MP", market.Initials)
	require.NotEmpty(t, market.PrimaryColor)
	require.NotEmpty(t, market.BackgroundColor)
	require.Equal(t, "company", market.Source)
	require.NotEmpty(t, market.ID)
}

func TestCreateMarket_SameNameTwice_IsIdempotent(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	first := postJSON(t, router, "/api/markets", map[string]string{"name": "Mercado Único"})
	require.Equal(t, http.StatusCreated, first.Code)
	var firstMarket models.Market
	require.NoError(t, json.Unmarshal(first.Body.Bytes(), &firstMarket))

	second := postJSON(t, router, "/api/markets", map[string]string{"name": "mercado único"})
	require.Equal(t, http.StatusCreated, second.Code)
	var secondMarket models.Market
	require.NoError(t, json.Unmarshal(second.Body.Bytes(), &secondMarket))

	require.Equal(t, firstMarket.ID, secondMarket.ID)
}

func TestCreateMarket_MissingName_ReturnsBadRequest(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	rec := postJSON(t, router, "/api/markets", map[string]string{"name": "   "})
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateMarket_InvalidSource_ReturnsBadRequest(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	rec := postJSON(t, router, "/api/markets", map[string]string{"name": "X", "source": "seed"})
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateMarket_BotSource_IsAccepted(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	rec := postJSON(t, router, "/api/markets", map[string]string{"name": "Loja Coletada", "source": "bot"})
	require.Equal(t, http.StatusCreated, rec.Code)

	var market models.Market
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &market))
	require.Equal(t, "bot", market.Source)
}
