package httpserver_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"priceuser/internal/httpserver"
	"priceuser/internal/models"
	"priceuser/internal/testutil"
)

func TestEconomySummary_ReturnsSeededTotalsAndChart(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	req := httptest.NewRequest(http.MethodGet, "/api/economy/summary", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var summary models.EconomySummary
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &summary))

	require.InDelta(t, 517.10, summary.TotalSaved, 0.001)
	require.Equal(t, "Maio de 2026", summary.MonthLabel)
	require.InDelta(t, 534.60, summary.MonthSpend, 0.001)
	require.InDelta(t, 112.30, summary.MonthSavings, 0.001)
	require.InDelta(t, 17.00, summary.MonthSavingsPercent, 0.001)

	require.Len(t, summary.MonthlyChart, 6)
	require.Equal(t, "Dez", summary.MonthlyChart[0].Label)
	require.False(t, summary.MonthlyChart[0].IsActive)
	require.Equal(t, "Mai", summary.MonthlyChart[5].Label)
	require.InDelta(t, 112, summary.MonthlyChart[5].Value, 0.001)
	require.True(t, summary.MonthlyChart[5].IsActive)
}
