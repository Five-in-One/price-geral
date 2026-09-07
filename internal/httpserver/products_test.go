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

func TestListProducts_SeedData_HasSourceSeed(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	req := httptest.NewRequest(http.MethodGet, "/api/products", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var products []models.Product
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &products))
	for _, p := range products {
		require.Equal(t, "seed", p.Source)
	}
}

func TestListProducts_SourceFilter(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	created := postJSON(t, router, "/api/products", map[string]any{
		"name": "Item da Empresa", "unit": "un", "category": "OUTROS",
		"price": 9.99, "market_name": "Loja Parceira X",
	})
	require.Equal(t, http.StatusCreated, created.Code)

	rec := doRequest(t, router, http.MethodGet, "/api/products?source=company")
	require.Equal(t, http.StatusOK, rec.Code)

	var products []models.Product
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &products))
	require.Len(t, products, 1)
	require.Equal(t, "Item da Empresa", products[0].Name)
	require.Equal(t, "company", products[0].Source)
}

func TestCreateProduct_WithNewMarketName_CreatesMarketAndProduct(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	rec := postJSON(t, router, "/api/products", map[string]any{
		"name": "Queijo Coalho", "unit": "kg", "category": "LATICÍNIOS",
		"price": 32.50, "market_name": "Fazenda do Sertão",
	})
	require.Equal(t, http.StatusCreated, rec.Code)

	var product models.Product
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &product))
	require.Equal(t, "Queijo Coalho", product.Name)
	require.Equal(t, "Fazenda do Sertão", product.DefaultMarketName)
	require.InDelta(t, 32.50, product.DefaultPrice, 0.001)
	require.Equal(t, "company", product.Source)

	marketsRec := doRequest(t, router, http.MethodGet, "/api/markets?all=true&search=sertão")
	var markets []models.MarketWithDistance
	require.NoError(t, json.Unmarshal(marketsRec.Body.Bytes(), &markets))
	require.Len(t, markets, 1)
	require.Equal(t, "company", markets[0].Source)
}

func TestCreateProduct_WithExistingMarketID(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	rec := postJSON(t, router, "/api/products", map[string]any{
		"name": "Iogurte Natural", "unit": "un", "category": "LATICÍNIOS",
		"price": 6.50, "market_id": "af",
	})
	require.Equal(t, http.StatusCreated, rec.Code)

	var product models.Product
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &product))
	require.Equal(t, "Atacado Família", product.DefaultMarketName)
}

func TestCreateProduct_UnknownMarketID_ReturnsBadRequest(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	rec := postJSON(t, router, "/api/products", map[string]any{
		"name": "X", "unit": "un", "category": "OUTROS",
		"price": 1.0, "market_id": "does-not-exist",
	})
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateProduct_ResubmittingSameMarketAndName_UpdatesPriceInstead(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	first := postJSON(t, router, "/api/products", map[string]any{
		"name": "Manga Espada", "unit": "kg", "category": "HORTIFRUTI",
		"price": 3.00, "market_id": "af",
	})
	require.Equal(t, http.StatusCreated, first.Code)
	var firstProduct models.Product
	require.NoError(t, json.Unmarshal(first.Body.Bytes(), &firstProduct))

	second := postJSON(t, router, "/api/products", map[string]any{
		"name": "manga espada", "unit": "kg", "category": "HORTIFRUTI",
		"price": 3.50, "market_id": "af",
	})
	require.Equal(t, http.StatusCreated, second.Code)
	var secondProduct models.Product
	require.NoError(t, json.Unmarshal(second.Body.Bytes(), &secondProduct))

	require.Equal(t, firstProduct.ID, secondProduct.ID)
	require.InDelta(t, 3.50, secondProduct.DefaultPrice, 0.001)
}

func TestCreateProduct_MissingRequiredFields_ReturnsBadRequest(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	rec := postJSON(t, router, "/api/products", map[string]any{
		"name": "", "unit": "un", "category": "OUTROS", "price": 1.0, "market_id": "af",
	})
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateProduct_NonPositivePrice_ReturnsBadRequest(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	rec := postJSON(t, router, "/api/products", map[string]any{
		"name": "X", "unit": "un", "category": "OUTROS", "price": 0, "market_id": "af",
	})
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateProduct_NoMarketGiven_ReturnsBadRequest(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	rec := postJSON(t, router, "/api/products", map[string]any{
		"name": "X", "unit": "un", "category": "OUTROS", "price": 1.0,
	})
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func doRequest(t *testing.T, router http.Handler, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}
