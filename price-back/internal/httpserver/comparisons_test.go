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

func doComparisonRequest(t *testing.T, router http.Handler, body models.CompareRequest) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/comparisons", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestCompare_TwoItems_RanksMarketsAndComputesDifference(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	// p1 (Banana prata) x2 + p7 (Arroz branco 5kg) x1.
	// af: 4.20*2 + 23.90*1 = 32.30 (cheapest)
	// se: 4.40*2 + 24.50*1 = 33.30
	// ms: 4.60*2 + 25.90*1 = 35.10
	// md: 4.79*2 + 26.90*1 = 36.48
	// hc: 4.79*2 + 27.90*1 = 37.48
	body := models.CompareRequest{Items: []models.CompareItem{
		{ProductID: "p1", Quantity: 2},
		{ProductID: "p7", Quantity: 1},
	}}

	rec := doComparisonRequest(t, router, body)
	require.Equal(t, http.StatusOK, rec.Code)

	var results []models.ComparisonResult
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &results))
	require.Len(t, results, 5)

	require.Equal(t, "af", results[0].Market.ID)
	require.InDelta(t, 32.30, results[0].TotalCost, 0.01)
	require.Equal(t, 3, results[0].MatchedItemsCount)
	require.InDelta(t, 0.0, results[0].Difference, 0.01)
	require.Len(t, results[0].Items, 2)

	require.Equal(t, "hc", results[4].Market.ID)
	require.InDelta(t, 37.48, results[4].TotalCost, 0.01)
	require.InDelta(t, 37.48-32.30, results[4].Difference, 0.01)

	// Strictly ascending order.
	for i := 1; i < len(results); i++ {
		require.LessOrEqual(t, results[i-1].TotalCost, results[i].TotalCost)
	}
}

func TestCompare_EmptyItems_ReturnsZeroedMarkets(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	rec := doComparisonRequest(t, router, models.CompareRequest{Items: []models.CompareItem{}})
	require.Equal(t, http.StatusOK, rec.Code)

	var results []models.ComparisonResult
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &results))
	require.Len(t, results, 5)
	for _, res := range results {
		require.Equal(t, 0.0, res.TotalCost)
		require.Equal(t, 0, res.MatchedItemsCount)
		require.Equal(t, 0.0, res.Difference)
		require.Empty(t, res.Items)
	}
}

func TestCompare_UnknownProduct_ReturnsBadRequest(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	rec := doComparisonRequest(t, router, models.CompareRequest{Items: []models.CompareItem{
		{ProductID: "does-not-exist", Quantity: 1},
	}})
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCompare_NonPositiveQuantity_ReturnsBadRequest(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	rec := doComparisonRequest(t, router, models.CompareRequest{Items: []models.CompareItem{
		{ProductID: "p1", Quantity: 0},
	}})
	require.Equal(t, http.StatusBadRequest, rec.Code)
}
