package httpserver_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"priceback/internal/httpserver"
	"priceback/internal/models"
	"priceback/internal/testutil"
)

// listing registers one market's offer of a product the way the bot does:
// with the fields the parser extracted from the name.
func listing(t *testing.T, router http.Handler, market, name string, price float64, extra map[string]any) models.Product {
	t.Helper()
	body := map[string]any{
		"name": name, "unit": "un", "category": "OUTROS", "price": price,
		"market_id": market, "source": "bot", "pack_count": 1,
	}
	for k, v := range extra {
		body[k] = v
	}
	rec := postJSON(t, router, "/api/products", body)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	return decodeProduct(t, rec.Body.Bytes())
}

func melitta500() map[string]any {
	return map[string]any{
		"brand": "Melitta", "generic_key": "cafe tradicional vacuo",
		"size_kind": "mass", "size_amount": 500, "size_label": "500 g",
	}
}

func catalogItems(t *testing.T, router http.Handler, query string) []models.CatalogItem {
	t.Helper()
	rec := doRequest(t, router, http.MethodGet, "/api/catalog/items"+query)
	require.Equal(t, http.StatusOK, rec.Code)
	var items []models.CatalogItem
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &items))
	return items
}

func pendingSuggestions(t *testing.T, router http.Handler) []models.CatalogSuggestion {
	t.Helper()
	rec := doRequest(t, router, http.MethodGet, "/api/catalog/suggestions")
	require.Equal(t, http.StatusOK, rec.Code)
	var out []models.CatalogSuggestion
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	return out
}

func TestCatalog_SameProductInTwoMarkets_SharesOneItem(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	a := listing(t, router, "af", "Café Melitta Tradicional Vácuo 500g", 28.90, melitta500())
	b := listing(t, router, "se", "Cafe Melitta Tradicional Vacuo 500g", 25.90, melitta500())

	require.NotEmpty(t, a.CatalogID)
	require.Equal(t, a.CatalogID, b.CatalogID, "same brand + generic key + size = same product")

	items := catalogItems(t, router, "")
	require.Len(t, items, 1)
	item := items[0]
	require.Equal(t, "Melitta", item.Brand)
	require.Equal(t, 2, item.MarketsCount)
	require.Len(t, item.Offers, 2)
	require.InDelta(t, 25.90, *item.MinPrice, 0.001)
	require.InDelta(t, 28.90, *item.MaxPrice, 0.001)
	require.Equal(t, "se", item.Offers[0].MarketID, "cheapest offer first")
	require.InDelta(t, 51.80, *item.Offers[0].UnitPrice, 0.001)
	require.Equal(t, "kg", item.Offers[0].UnitPriceUnit)

	one := doRequest(t, router, http.MethodGet, "/api/catalog/items/"+item.ID)
	require.Equal(t, http.StatusOK, one.Code)
}

func TestCatalog_DifferentSizesAreDifferentItems(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	listing(t, router, "af", "Café Melitta 500g", 28.90, melitta500())
	small := melitta500()
	small["size_amount"], small["size_label"] = 250, "250 g"
	listing(t, router, "se", "Café Melitta 250g", 12.90, small)

	require.Len(t, catalogItems(t, router, ""), 2, "a 250 g pack is not the 500 g pack")
	require.Empty(t, pendingSuggestions(t, router), "different sizes are never even suggested")
}

func TestCatalog_ListingWithoutParsedFields_IsNotLinked(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	rec := postJSON(t, router, "/api/products", map[string]any{
		"name": "Produto Da Página Da Empresa", "unit": "un", "category": "OUTROS", "price": 5, "market_id": "af",
	})
	require.Equal(t, http.StatusCreated, rec.Code)
	require.Empty(t, decodeProduct(t, rec.Body.Bytes()).CatalogID)
	require.Empty(t, catalogItems(t, router, ""))
}

func TestCatalog_ResubmittingAListing_DoesNotDuplicateItems(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	first := listing(t, router, "af", "Café Melitta 500g", 28.90, melitta500())
	again := listing(t, router, "af", "Café Melitta 500g", 29.50, melitta500())

	require.Equal(t, first.CatalogID, again.CatalogID)
	items := catalogItems(t, router, "")
	require.Len(t, items, 1)
	require.Len(t, items[0].Offers, 1)
	require.InDelta(t, 29.50, items[0].Offers[0].Price, 0.001)
}

func TestCatalog_ListFilters(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	listing(t, router, "af", "Café Melitta 500g", 28.90, melitta500())
	listing(t, router, "se", "Café Melitta 500g", 25.90, melitta500())
	listing(t, router, "af", "Arroz Camil 5kg", 22.69, map[string]any{
		"brand": "Camil", "generic_key": "arroz branco tipo1", "size_kind": "mass", "size_amount": 5000,
	})

	require.Len(t, catalogItems(t, router, ""), 2)

	shared := catalogItems(t, router, "?min_markets=2")
	require.Len(t, shared, 1)
	require.Equal(t, "Melitta", shared[0].Brand)

	found := catalogItems(t, router, "?search=camil")
	require.Len(t, found, 1)
	require.Equal(t, "Camil", found[0].Brand)

	require.Equal(t, http.StatusBadRequest, doRequest(t, router, http.MethodGet, "/api/catalog/items?min_markets=x").Code)
	require.Equal(t, http.StatusNotFound, doRequest(t, router, http.MethodGet, "/api/catalog/items/nao-existe").Code)
}

// --- review queue -------------------------------------------------------------

func feijao(brand string) map[string]any {
	m := map[string]any{"generic_key": "carioca feijao tipo1", "size_kind": "mass", "size_amount": 1000}
	if brand != "" {
		m["brand"] = brand
	}
	return m
}

func TestCatalog_BrandMissingOnOneSide_QueuesASuggestionInsteadOfMerging(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	a := listing(t, router, "af", "Feijão Carioca Tipo 1 1kg", 7.49, feijao(""))
	b := listing(t, router, "se", "Feijão Carioca Sofia Tipo 1 1kg", 8.39, feijao("Sofia"))

	require.NotEqual(t, a.CatalogID, b.CatalogID, "never merged automatically")
	require.Len(t, catalogItems(t, router, ""), 2)

	queue := pendingSuggestions(t, router)
	require.Len(t, queue, 1)
	require.Equal(t, "brand-missing", queue[0].Reason)
	require.Equal(t, b.ID, queue[0].Product.ProductID)
	require.Equal(t, a.CatalogID, queue[0].Item.ID)
	require.Len(t, queue[0].Item.Offers, 1, "the reviewer sees what the candidate item already holds")
}

func TestCatalog_AcceptSuggestion_MergesAndSurvivesRecollection(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	a := listing(t, router, "af", "Feijão Carioca Tipo 1 1kg", 7.49, feijao(""))
	b := listing(t, router, "se", "Feijão Carioca Sofia Tipo 1 1kg", 8.39, feijao("Sofia"))
	suggestion := pendingSuggestions(t, router)[0]

	rec := doRequest(t, router, http.MethodPost, fmt.Sprintf("/api/catalog/suggestions/%d/accept", suggestion.ID))
	require.Equal(t, http.StatusNoContent, rec.Code)

	items := catalogItems(t, router, "")
	require.Len(t, items, 1, "b's own item became empty and was removed")
	require.Equal(t, a.CatalogID, items[0].ID)
	require.Equal(t, 2, items[0].MarketsCount)
	require.Empty(t, pendingSuggestions(t, router))

	// the bot collects the same product again: a person's decision must stick
	again := listing(t, router, "se", "Feijão Carioca Sofia Tipo 1 1kg", 8.49, feijao("Sofia"))
	require.Equal(t, a.CatalogID, again.CatalogID)
	require.Equal(t, b.ID, again.ID)
	require.Len(t, catalogItems(t, router, ""), 1)

	// a second decision on the same suggestion is a conflict
	require.Equal(t, http.StatusConflict,
		doRequest(t, router, http.MethodPost, fmt.Sprintf("/api/catalog/suggestions/%d/reject", suggestion.ID)).Code)
}

func TestCatalog_RejectSuggestion_KeepsThemApartAndIsFinal(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	listing(t, router, "af", "Feijão Carioca Tipo 1 1kg", 7.49, feijao(""))
	listing(t, router, "se", "Feijão Carioca Sofia Tipo 1 1kg", 8.39, feijao("Sofia"))
	suggestion := pendingSuggestions(t, router)[0]

	require.Equal(t, http.StatusNoContent,
		doRequest(t, router, http.MethodPost, fmt.Sprintf("/api/catalog/suggestions/%d/reject", suggestion.ID)).Code)

	require.Empty(t, pendingSuggestions(t, router))
	require.Len(t, catalogItems(t, router, ""), 2)

	rec := doRequest(t, router, http.MethodGet, "/api/catalog/suggestions?status=rejected")
	var rejected []models.CatalogSuggestion
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &rejected))
	require.Len(t, rejected, 1)

	// re-running the matcher never brings a rejected pair back
	require.Equal(t, http.StatusOK, doRequest(t, router, http.MethodPost, "/api/catalog/match?all=true").Code)
	require.Empty(t, pendingSuggestions(t, router))
}

func TestCatalog_NearlyIdenticalNames_AreSuggested(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	// "Corona Cero 330ml 0% Alc" vs "Corona Zero Álcool 330ml": same beer, written two ways
	beer := func(key string) map[string]any {
		return map[string]any{"brand": "Corona", "generic_key": key, "size_kind": "volume", "size_amount": 330}
	}
	listing(t, router, "af", "Cerveja Corona Zero Álcool 330ml", 7.49, beer("alcool cerveja zero"))
	listing(t, router, "se", "Cerveja Corona Cero 330ml Ln 0% Alc", 6.99, beer("0% alcool cerveja zero"))

	queue := pendingSuggestions(t, router)
	require.Len(t, queue, 1)
	require.Equal(t, "similar-name", queue[0].Reason)
	require.Greater(t, queue[0].Score, 0.85)
}

func TestCatalog_VariantsOfTheSameBrandAndSize_AreNotSuggested(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	// same brand, same size, but Zero is a different product from the regular beer
	beer := func(key string) map[string]any {
		return map[string]any{"brand": "Heineken", "generic_key": key, "size_kind": "volume", "size_amount": 330}
	}
	listing(t, router, "af", "Cerveja Heineken 330ml", 6.79, beer("cerveja"))
	listing(t, router, "af", "Cerveja Heineken Zero 330ml", 7.49, beer("cerveja zero"))

	require.Len(t, catalogItems(t, router, ""), 2)
	require.Empty(t, pendingSuggestions(t, router), "similar-looking variants must not pollute the queue")
}

func TestCatalog_BrandMissingButDifferentSize_IsNotSuggested(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	listing(t, router, "af", "Feijão Carioca Tipo 1 1kg", 7.49, feijao(""))
	big := feijao("Sofia")
	big["size_amount"] = 2000
	listing(t, router, "se", "Feijão Carioca Sofia Tipo 1 2kg", 15.00, big)

	require.Empty(t, pendingSuggestions(t, router))
}

func TestCatalog_DifferentBrandsAreNeverSuggested(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	listing(t, router, "af", "Feijão Carioca Kicaldo Tipo 1 1kg", 7.49, feijao("Kicaldo"))
	listing(t, router, "se", "Feijão Carioca Sofia Tipo 1 1kg", 8.39, feijao("Sofia"))

	// equivalent products, not the same product: that is a different feature
	require.Len(t, catalogItems(t, router, ""), 2)
	require.Empty(t, pendingSuggestions(t, router))
}

func TestCatalog_DecideUnknownSuggestion(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	require.Equal(t, http.StatusNotFound, doRequest(t, router, http.MethodPost, "/api/catalog/suggestions/999/accept").Code)
	require.Equal(t, http.StatusNotFound, doRequest(t, router, http.MethodPost, "/api/catalog/suggestions/999/reject").Code)
	require.Equal(t, http.StatusBadRequest, doRequest(t, router, http.MethodPost, "/api/catalog/suggestions/abc/accept").Code)
	require.Equal(t, http.StatusBadRequest, doRequest(t, router, http.MethodGet, "/api/catalog/suggestions?status=bogus").Code)
}

// --- backfill --------------------------------------------------------------------

func TestCatalog_MatchBackfillsListingsThatPredateTheCatalog(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	listing(t, router, "af", "Café Melitta 500g", 28.90, melitta500())
	listing(t, router, "se", "Café Melitta 500g", 25.90, melitta500())
	postJSON(t, router, "/api/products", map[string]any{
		"name": "Sem Campos", "unit": "un", "category": "OUTROS", "price": 5, "market_id": "af",
	})

	// simulate the state before the catalog existed
	_, err := pool.Exec(t.Context(), `UPDATE products SET catalog_id = NULL`)
	require.NoError(t, err)
	_, err = pool.Exec(t.Context(), `DELETE FROM catalog_items`)
	require.NoError(t, err)
	require.Empty(t, catalogItems(t, router, ""))

	rec := doRequest(t, router, http.MethodPost, "/api/catalog/match")
	require.Equal(t, http.StatusOK, rec.Code)
	var summary models.MatchSummary
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &summary))
	require.Equal(t, 2, summary.Processed)
	require.Equal(t, 2, summary.Linked)
	require.Equal(t, 1, summary.ItemsCreated, "both listings are the same product")
	// the fixture's 12 seed products plus "Sem Campos" have no generic key to identify them by
	require.GreaterOrEqual(t, summary.Skipped, 1, "listings with no generic key can't be identified")

	items := catalogItems(t, router, "")
	require.Len(t, items, 1)
	require.Equal(t, 2, items[0].MarketsCount)

	// idempotent
	rec = doRequest(t, router, http.MethodPost, "/api/catalog/match")
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &summary))
	require.Zero(t, summary.Processed)
}

func TestCatalog_PatchingParsedFieldsLinksTheListing(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	created := postJSON(t, router, "/api/products", map[string]any{
		"name": "Café Melitta 500g", "unit": "un", "category": "GRÃOS", "price": 28.90, "market_id": "af",
	})
	id := decodeProduct(t, created.Body.Bytes()).ID
	require.Empty(t, catalogItems(t, router, ""))

	rec := patchJSON(t, router, "/api/products/"+id, map[string]any{
		"brand": "Melitta", "generic_key": "cafe tradicional vacuo", "size_kind": "mass", "size_amount": 500,
	})
	require.Equal(t, http.StatusOK, rec.Code)
	require.NotEmpty(t, decodeProduct(t, rec.Body.Bytes()).CatalogID)
	require.Len(t, catalogItems(t, router, ""), 1)
}

func TestCatalog_ChangedIdentityMovesTheListingAndDropsTheEmptyItem(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	p := listing(t, router, "af", "Café X 500g", 10, map[string]any{
		"brand": "Melitta", "generic_key": "cafe", "size_kind": "mass", "size_amount": 500,
	})
	before := p.CatalogID

	// the parser improved and now reads the brand differently
	rec := patchJSON(t, router, "/api/products/"+p.ID, map[string]any{"brand": "Pilão"})
	require.Equal(t, http.StatusOK, rec.Code)
	after := decodeProduct(t, rec.Body.Bytes()).CatalogID

	require.NotEqual(t, before, after)
	items := catalogItems(t, router, "")
	require.Len(t, items, 1, "the old, now empty, item is gone")
	require.Equal(t, "Pilão", items[0].Brand)
}
