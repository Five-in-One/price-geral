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

func TestCreateMarket_WithAddress_StoresAddressFields(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	rec := postJSON(t, router, "/api/markets", map[string]string{
		"name":   "Mercado Endereçado",
		"source": "bot",
		"link":   "https://www.mercado-enderecado.com.br",
		"cep":    "64000-000",
		"street": "Rua Principal",
		"number": "123",
		"city":   "Teresina",
		"state":  "PI",
	})
	require.Equal(t, http.StatusCreated, rec.Code)

	var market models.Market
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &market))
	require.Equal(t, []string{"https://www.mercado-enderecado.com.br"}, market.Links)
	require.Equal(t, "64000-000", market.CEP)
	require.Equal(t, "Rua Principal", market.Street)
	require.Equal(t, "123", market.Number)
	require.Equal(t, "Teresina", market.City)
	require.Equal(t, "PI", market.State)
}

func TestCreateMarket_SameNameDifferentLink_AccumulatesLinks(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	first := postJSON(t, router, "/api/markets", map[string]string{
		"name": "Rede Multi Loja", "source": "bot", "link": "https://www.loja1.com.br",
	})
	require.Equal(t, http.StatusCreated, first.Code)

	second := postJSON(t, router, "/api/markets", map[string]string{
		"name": "Rede Multi Loja", "source": "bot", "link": "https://www.loja2.com.br",
	})
	require.Equal(t, http.StatusCreated, second.Code)

	// Resubmitting the exact same link a second time must not duplicate it.
	third := postJSON(t, router, "/api/markets", map[string]string{
		"name": "Rede Multi Loja", "source": "bot", "link": "https://www.loja2.com.br",
	})
	require.Equal(t, http.StatusCreated, third.Code)

	var market models.Market
	require.NoError(t, json.Unmarshal(third.Body.Bytes(), &market))
	require.Equal(t, []string{"https://www.loja1.com.br", "https://www.loja2.com.br"}, market.Links)
}

func TestCreateMarket_SameNameTwice_MergesAddressWithoutBlankingIt(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	first := postJSON(t, router, "/api/markets", map[string]string{
		"name": "Mercado Progressivo", "source": "bot", "cep": "64000-000", "city": "Teresina",
	})
	require.Equal(t, http.StatusCreated, first.Code)

	// Resubmitting with a blank cep/city (e.g. the admin only had the URL
	// this time) must not erase what was already stored.
	second := postJSON(t, router, "/api/markets", map[string]string{
		"name": "Mercado Progressivo", "source": "bot", "link": "https://www.progressivo.com.br",
	})
	require.Equal(t, http.StatusCreated, second.Code)

	var market models.Market
	require.NoError(t, json.Unmarshal(second.Body.Bytes(), &market))
	require.Equal(t, []string{"https://www.progressivo.com.br"}, market.Links)
	require.Equal(t, "64000-000", market.CEP, "earlier cep must survive a resubmit that omits it")
	require.Equal(t, "Teresina", market.City, "earlier city must survive a resubmit that omits it")
}

func TestAddMarketLink_AppendsToList(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	rec := postJSON(t, router, "/api/markets/af/links", map[string]string{"url": "https://www.af.com.br"})
	require.Equal(t, http.StatusOK, rec.Code)

	var market models.Market
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &market))
	require.Equal(t, []string{"https://www.af.com.br"}, market.Links)
}

func TestAddMarketLink_SameURLTwice_DoesNotDuplicate(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	postJSON(t, router, "/api/markets/af/links", map[string]string{"url": "https://www.af.com.br"})
	rec := postJSON(t, router, "/api/markets/af/links", map[string]string{"url": "https://www.af.com.br"})

	require.Equal(t, http.StatusOK, rec.Code)
	var market models.Market
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &market))
	require.Equal(t, []string{"https://www.af.com.br"}, market.Links)
}

func TestAddMarketLink_EmptyURL_ReturnsBadRequest(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	rec := postJSON(t, router, "/api/markets/af/links", map[string]string{"url": "  "})
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAddMarketLink_UnknownMarket_ReturnsNotFound(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	rec := postJSON(t, router, "/api/markets/does-not-exist/links", map[string]string{"url": "https://x.com"})
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestRemoveMarketLink_RemovesIt(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	postJSON(t, router, "/api/markets/af/links", map[string]string{"url": "https://www.af.com.br"})

	rec := doRequest(t, router, http.MethodDelete, "/api/markets/af/links?url=https://www.af.com.br")
	require.Equal(t, http.StatusOK, rec.Code)

	var market models.Market
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &market))
	require.Empty(t, market.Links)
}

func TestRemoveMarketLink_UnknownMarket_ReturnsNotFound(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	rec := doRequest(t, router, http.MethodDelete, "/api/markets/does-not-exist/links?url=https://x.com")
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestRemoveMarketLink_MissingURLParam_ReturnsBadRequest(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	rec := doRequest(t, router, http.MethodDelete, "/api/markets/af/links")
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateMarket_ChangesNameAndColors(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	rec := patchJSON(t, router, "/api/markets/af", map[string]any{
		"name": "Atacado Família Ltda", "primary_color": "#000000", "background_color": "#FFFFFF",
	})
	require.Equal(t, http.StatusOK, rec.Code)

	var market models.Market
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &market))
	require.Equal(t, "Atacado Família Ltda", market.Name)
	require.Equal(t, "#000000", market.PrimaryColor)
	require.Equal(t, "#FFFFFF", market.BackgroundColor)
	require.Equal(t, "AF", market.Initials, "initials untouched when not provided")
}

func TestUpdateMarket_EmptyName_ReturnsBadRequest(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	rec := patchJSON(t, router, "/api/markets/af", map[string]any{"name": "   "})
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateMarket_UnknownID_ReturnsNotFound(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	rec := patchJSON(t, router, "/api/markets/does-not-exist", map[string]any{"name": "X"})
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestDeleteMarket_RemovesItAndItsProducts(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	rec := doRequest(t, router, http.MethodDelete, "/api/markets/af")
	require.Equal(t, http.StatusNoContent, rec.Code)

	marketsRec := doRequest(t, router, http.MethodGet, "/api/markets?all=true")
	var markets []models.MarketWithDistance
	require.NoError(t, json.Unmarshal(marketsRec.Body.Bytes(), &markets))
	require.Len(t, markets, 4)

	productsRec := doRequest(t, router, http.MethodGet, "/api/products?market_id=af")
	var products []models.Product
	require.NoError(t, json.Unmarshal(productsRec.Body.Bytes(), &products))
	require.Empty(t, products, "products whose default market was deleted should go with it")
}

func TestDeleteMarket_UnknownID_ReturnsNotFound(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	rec := doRequest(t, router, http.MethodDelete, "/api/markets/does-not-exist")
	require.Equal(t, http.StatusNotFound, rec.Code)
}
