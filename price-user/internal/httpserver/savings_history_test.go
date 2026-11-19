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

func TestListSavingsHistory_ReturnsSeededItemsNewestFirst(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool, "test-secret")

	req := httptest.NewRequest(http.MethodGet, "/api/savings-history", nil)
	req.Header.Set("Authorization", "Bearer "+tokenFor(t, "user-1"))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var items []models.SavingsHistoryItem
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &items))
	require.Len(t, items, 3)

	require.Equal(t, "24 MAI", items[0].DateLabel)
	require.Equal(t, "Atacado Família", items[0].MarketName)
	require.Equal(t, 12, items[0].ItemsCount)
	require.InDelta(t, 187.40, items[0].TotalCost, 0.001)
	require.InDelta(t, 28.60, items[0].Savings, 0.001)

	require.Equal(t, "17 MAI", items[1].DateLabel)
	require.Equal(t, "10 MAI", items[2].DateLabel)
}
