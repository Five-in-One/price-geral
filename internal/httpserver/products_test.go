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

func TestListProducts_NoFilters_ReturnsAllOrderedByName(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	req := httptest.NewRequest(http.MethodGet, "/api/products", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var products []models.Product
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &products))
	require.Len(t, products, 12)

	for i := 1; i < len(products); i++ {
		require.LessOrEqual(t, products[i-1].Name, products[i].Name)
	}
}

func TestListProducts_CategoryFilter(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	req := httptest.NewRequest(http.MethodGet, "/api/products?category=GR%C3%83OS", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var products []models.Product
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &products))
	require.Len(t, products, 4)
	for _, p := range products {
		require.Equal(t, "GRÃOS", p.Category)
	}
}

func TestListProducts_SearchFilter_IsCaseInsensitive(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	req := httptest.NewRequest(http.MethodGet, "/api/products?search=banana", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var products []models.Product
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &products))
	require.Len(t, products, 1)
	require.Equal(t, "Banana prata", products[0].Name)
	require.Equal(t, "Hortifruti Central", products[0].DefaultMarketName)
	require.InDelta(t, 4.79, products[0].DefaultPrice, 0.001)
}

func TestListProducts_SortByPriceAsc(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	req := httptest.NewRequest(http.MethodGet, "/api/products?sort=price_asc", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var products []models.Product
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &products))
	require.Len(t, products, 12)

	for i := 1; i < len(products); i++ {
		require.LessOrEqual(t, products[i-1].DefaultPrice, products[i].DefaultPrice)
	}
	require.Equal(t, "Alface crespa", products[0].Name) // cheapest at 2.49
}

func TestListProducts_NoMatches_ReturnsEmptyArrayNotNull(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	req := httptest.NewRequest(http.MethodGet, "/api/products?search=xyz-nope", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "[]\n", rec.Body.String())
}
