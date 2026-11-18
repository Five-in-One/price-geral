package httpserver_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"priceback/internal/httpserver"
	"priceback/internal/models"
	"priceback/internal/testutil"
)

func searchProducts(t *testing.T, router http.Handler, query string) models.ProductPage {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/products/search"+query, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var page models.ProductPage
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &page))
	return page
}

func names(page models.ProductPage) []string {
	out := make([]string, len(page.Items))
	for i, p := range page.Items {
		out[i] = p.Name
	}
	return out
}

func TestSearchProducts_DefaultsToCheapestFirstPagedBy20(t *testing.T) {
	router := httpserver.NewRouter(testutil.NewTestPool(t))

	page := searchProducts(t, router, "")

	require.Equal(t, 12, page.Total)
	require.Equal(t, 20, page.Limit)
	require.Len(t, page.Items, 12)
	require.False(t, page.HasMore)
	require.Equal(t, "Alface crespa", page.Items[0].Name)
	for i := 1; i < len(page.Items); i++ {
		require.LessOrEqual(t, page.Items[i-1].DefaultPrice, page.Items[i].DefaultPrice)
	}
}

func TestSearchProducts_PagesWithLimitAndOffset(t *testing.T) {
	router := httpserver.NewRouter(testutil.NewTestPool(t))

	first := searchProducts(t, router, "?limit=5")
	require.Len(t, first.Items, 5)
	require.Equal(t, 12, first.Total)
	require.True(t, first.HasMore)

	last := searchProducts(t, router, "?limit=5&offset=10")
	require.Len(t, last.Items, 2)
	require.False(t, last.HasMore)

	// Every row shows up exactly once across the pages.
	seen := map[string]bool{}
	for offset := 0; offset < 12; offset += 5 {
		page := searchProducts(t, router, "?limit=5&offset="+strconv.Itoa(offset))
		for _, p := range page.Items {
			require.False(t, seen[p.ID], "duplicate %s", p.ID)
			seen[p.ID] = true
		}
	}
	require.Len(t, seen, 12)

	past := searchProducts(t, router, "?limit=5&offset=40")
	require.Empty(t, past.Items)
	require.Equal(t, 12, past.Total)
	require.False(t, past.HasMore)
}

func TestSearchProducts_SearchIgnoresAccentsAndMatchesEveryWord(t *testing.T) {
	router := httpserver.NewRouter(testutil.NewTestPool(t))

	require.Equal(t, []string{"Feijão carioca 1kg"}, names(searchProducts(t, router, "?search=FEIJAO%201kg")))
	require.Equal(t, []string{"Maçã gala"}, names(searchProducts(t, router, "?search=maca")))
	require.Empty(t, searchProducts(t, router, "?search=feijao%20preto").Items)
	// LIKE wildcards are literal text.
	require.Empty(t, searchProducts(t, router, "?search=%25").Items)
}

func TestSearchProducts_CategoryAndSort(t *testing.T) {
	router := httpserver.NewRouter(testutil.NewTestPool(t))

	page := searchProducts(t, router, "?category=gr%C3%A3os&sort=price_desc")
	require.Equal(t, 4, page.Total)
	require.Equal(t, "Arroz branco 5kg", page.Items[0].Name)

	byName := searchProducts(t, router, "?sort=name_asc&limit=3")
	require.Equal(t, []string{"Alface crespa", "Arroz branco 5kg", "Banana prata"}, names(byName))
}

func TestSearchProducts_MarketAndPriceFilters(t *testing.T) {
	router := httpserver.NewRouter(testutil.NewTestPool(t))

	require.Equal(t, 5, searchProducts(t, router, "?market_id=hc").Total)
	require.Equal(t, 12, searchProducts(t, router, "?market_id=hc,af").Total)

	page := searchProducts(t, router, "?min_price=4&max_price=5")
	require.ElementsMatch(t, []string{"Banana prata", "Batata inglesa", "Leite integral 1L"}, names(page))
}

func TestSearchProducts_RadiusKeepsOnlyNearbyMarkets(t *testing.T) {
	router := httpserver.NewRouter(testutil.NewTestPool(t))

	// Standing at Hortifruti Central: 300 m reaches no other market.
	page := searchProducts(t, router, "?lat=-5.077488&lng=-42.8016&radius_km=0.3")
	require.Equal(t, 5, page.Total)
	for _, p := range page.Items {
		require.Equal(t, "Hortifruti Central", p.DefaultMarketName)
	}

	// The radius narrows an explicit market filter instead of widening it.
	require.Equal(t, 0, searchProducts(t, router, "?market_id=af&lat=-5.077488&lng=-42.8016&radius_km=0.3").Total)
}

func TestSearchProducts_GroupsOffersOfOneCatalogItem(t *testing.T) {
	pool := testutil.NewTestPool(t)
	ctx := context.Background()
	_, err := pool.Exec(ctx, `
		INSERT INTO catalog_items (id, name, category, generic_key) VALUES ('cat-banana', 'Banana prata', 'HORTIFRUTI', 'banana prata');
		INSERT INTO products (id, name, unit, category, default_market_id, catalog_id)
			VALUES ('p13', 'Banana Prata Kg', 'kg', 'HORTIFRUTI', 'se', 'cat-banana');
		INSERT INTO market_products (market_id, product_id, price) VALUES ('se', 'p13', 3.99);
		UPDATE products SET catalog_id = 'cat-banana' WHERE id = 'p1';
	`)
	require.NoError(t, err)
	router := httpserver.NewRouter(pool)

	page := searchProducts(t, router, "?search=banana")
	require.Len(t, page.Items, 1)
	require.Equal(t, 1, page.Total)
	require.Equal(t, "p13", page.Items[0].ID)
	require.Equal(t, 3.99, page.Items[0].DefaultPrice)
	require.Equal(t, "Super Economia", page.Items[0].DefaultMarketName)
	require.Equal(t, 2, page.Items[0].MarketsCount)

	// 13 listings, 12 rows.
	require.Equal(t, 12, searchProducts(t, router, "").Total)
}

func TestSearchProducts_RejectsInvalidParameters(t *testing.T) {
	router := httpserver.NewRouter(testutil.NewTestPool(t))

	for _, query := range []string{
		"?sort=random",
		"?limit=0",
		"?limit=101",
		"?offset=-1",
		"?min_price=abc",
		"?min_price=10&max_price=5",
		"?radius_km=5",
		"?lat=-5&lng=-42&radius_km=0",
	} {
		req := httptest.NewRequest(http.MethodGet, "/api/products/search"+query, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		require.Equal(t, http.StatusBadRequest, rec.Code, query)
	}
}

func TestProductCategories_CountsRowsUnderTheOtherFilters(t *testing.T) {
	router := httpserver.NewRouter(testutil.NewTestPool(t))

	req := httptest.NewRequest(http.MethodGet, "/api/products/categories?category=CARNES", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var counts []models.CategoryCount
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &counts))
	require.Equal(t, []models.CategoryCount{
		{Category: "HORTIFRUTI", Count: 6},
		{Category: "GRÃOS", Count: 4},
		{Category: "CARNES", Count: 1},
		{Category: "LATICÍNIOS", Count: 1},
	}, counts)
}
