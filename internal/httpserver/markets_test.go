package httpserver_test

import (
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
