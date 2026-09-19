package httpserver_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"priceback/internal/httpserver"
	"priceback/internal/models"
	"priceback/internal/testutil"
)

func compareList(t *testing.T, router http.Handler, body map[string]any) models.ListComparison {
	t.Helper()
	rec := postJSON(t, router, "/api/comparisons/list", body)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var out models.ListComparison
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	return out
}

func marketByID(t *testing.T, c models.ListComparison, id string) models.MarketComparison {
	t.Helper()
	for _, m := range c.Markets {
		if m.Market.ID == id {
			return m
		}
	}
	require.FailNow(t, "market not in comparison", id)
	return models.MarketComparison{}
}

func lineFor(m models.MarketComparison, catalogID string) *models.ListLine {
	for i := range m.Lines {
		if m.Lines[i].CatalogID == catalogID {
			return &m.Lines[i]
		}
	}
	return nil
}

func arroz(brand string, kg float64) map[string]any {
	return map[string]any{
		"brand": brand, "generic_key": "arroz branco tipo1",
		"size_kind": "mass", "size_amount": kg * 1000, "size_label": "5 kg",
	}
}

func item(catalogID string, qty int) map[string]any {
	return map[string]any{"catalog_id": catalogID, "quantity": qty}
}

func TestCompareList_SameProductAtTwoMarkets_RanksByTotal(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	a := listing(t, router, "af", "Café Melitta 500g", 28.90, melitta500())
	listing(t, router, "se", "Café Melitta 500g", 25.90, melitta500())

	c := compareList(t, router, map[string]any{"items": []any{item(a.CatalogID, 2)}})

	require.Equal(t, "se", c.Markets[0].Market.ID)
	require.InDelta(t, 51.80, c.Markets[0].TotalCost, 0.001)
	require.True(t, c.Markets[0].IsComplete)
	require.Equal(t, "af", c.Markets[1].Market.ID)
	require.InDelta(t, 57.80, c.Markets[1].TotalCost, 0.001)
	require.InDelta(t, 6.00, c.Markets[1].Difference, 0.001)

	// the other three fixture markets don't sell it: ranked after, incomplete
	require.Len(t, c.Markets, 5)
	for _, m := range c.Markets[2:] {
		require.False(t, m.IsComplete)
		require.Len(t, m.Missing, 1)
		require.Zero(t, m.Difference, "incomplete markets can't be compared with complete ones")
	}

	require.Equal(t, "se", c.BestSingleMarketID)
	require.InDelta(t, 6.00, c.MaxSavings, 0.001)
	require.InDelta(t, 10.38, c.MaxSavingsPercent, 0.01)

	cheap := lineFor(c.Markets[0], a.CatalogID)
	require.True(t, cheap.IsCheapest)
	require.False(t, lineFor(c.Markets[1], a.CatalogID).IsCheapest)
	require.InDelta(t, 51.80, *cheap.UnitPrice, 0.001)
	require.Equal(t, "kg", cheap.UnitPriceUnit)
	require.InDelta(t, 51.80, cheap.LineTotal, 0.001)

	require.Equal(t, "se", c.Items[0].CheapestMarketID)
	require.InDelta(t, 25.90, *c.Items[0].CheapestPrice, 0.001)
}

func TestCompareList_MissingItem_MarketRanksAfterCompleteOnesAndReportsIt(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	cafe := listing(t, router, "af", "Café Melitta 500g", 28.90, melitta500())
	listing(t, router, "se", "Café Melitta 500g", 25.90, melitta500())
	rice := listing(t, router, "af", "Arroz Camil 5kg", 22.69, arroz("Camil", 5))

	c := compareList(t, router, map[string]any{"items": []any{item(cafe.CatalogID, 1), item(rice.CatalogID, 1)}})

	require.Equal(t, "af", c.Markets[0].Market.ID)
	require.True(t, c.Markets[0].IsComplete)
	require.InDelta(t, 51.59, c.Markets[0].TotalCost, 0.001)

	se := marketByID(t, c, "se")
	require.False(t, se.IsComplete)
	require.Equal(t, 1, se.FoundItems)
	require.Equal(t, 2, se.TotalItems)
	require.Len(t, se.Missing, 1)
	require.Equal(t, rice.CatalogID, se.Missing[0].CatalogID)
	require.Equal(t, "se", c.Markets[1].Market.ID, "the one with more items comes before the empty ones")
	require.InDelta(t, 25.90, se.TotalCost, 0.001, "a partial total is never presented as comparable")
}

func TestCompareList_BestCombination_BuysEachItemWhereItIsCheapest(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	cafe := listing(t, router, "af", "Café Melitta 500g", 28.90, melitta500())
	listing(t, router, "se", "Café Melitta 500g", 25.90, melitta500())
	rice := listing(t, router, "af", "Arroz Camil 5kg", 22.69, arroz("Camil", 5))

	c := compareList(t, router, map[string]any{"items": []any{item(cafe.CatalogID, 1), item(rice.CatalogID, 1)}})

	combo := c.BestCombination
	require.NotNil(t, combo)
	require.InDelta(t, 25.90+22.69, combo.TotalCost, 0.001)
	require.Len(t, combo.Stops, 2)
	require.NotNil(t, combo.SavingsVsBestSingle)
	require.InDelta(t, 3.00, *combo.SavingsVsBestSingle, 0.001, "af alone costs 51.59")
}

func TestCompareList_Combination_DoesNotAddAStopForATie(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	a := listing(t, router, "af", "Café Melitta 500g", 10, melitta500())
	other := map[string]any{"brand": "Pilão", "generic_key": "cafe", "size_kind": "mass", "size_amount": 500}
	b := listing(t, router, "af", "Café Pilão 500g", 5, other)
	listing(t, router, "se", "Café Pilão 500g", 5, other)

	c := compareList(t, router, map[string]any{"items": []any{item(a.CatalogID, 1), item(b.CatalogID, 1)}})

	require.Len(t, c.BestCombination.Stops, 1, "same price at af and se: stay at af, which is already on the route")
	require.Equal(t, "af", c.BestCombination.Stops[0].Market.ID)
}

func TestCompareList_Equivalents_FillGapsOnlyWhenAllowed(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	camil := listing(t, router, "af", "Arroz Branco Camil Tipo 1 5kg", 22.69, arroz("Camil", 5))
	sofia := listing(t, router, "se", "Arroz Branco Sofia Tipo 1 5kg", 21.90, arroz("Sofia", 5))
	items := []any{item(camil.CatalogID, 1)}

	strict := compareList(t, router, map[string]any{"items": items})
	require.False(t, marketByID(t, strict, "se").IsComplete)
	require.False(t, strict.AllowEquivalents)

	flexible := compareList(t, router, map[string]any{"items": items, "allow_equivalents": true})
	se := marketByID(t, flexible, "se")
	require.True(t, se.IsComplete)
	require.Equal(t, 1, se.SubstitutesCount)
	line := lineFor(se, camil.CatalogID)
	require.True(t, line.IsSubstitute)
	require.Equal(t, "Sofia", line.Brand)
	require.Equal(t, sofia.ID, line.ProductID)
	require.Equal(t, camil.CatalogID, line.CatalogID, "the line still belongs to the item the user asked for")
	require.InDelta(t, 21.90, line.Price, 0.001)

	// se is now the cheapest market for this list
	require.Equal(t, "se", flexible.BestSingleMarketID)
	require.True(t, line.IsCheapest)
	require.False(t, lineFor(marketByID(t, flexible, "af"), camil.CatalogID).IsSubstitute)
}

func TestCompareList_RequestedBrandWinsOverACheaperSubstitute(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	camil := listing(t, router, "af", "Arroz Branco Camil Tipo 1 5kg", 22.69, arroz("Camil", 5))
	listing(t, router, "se", "Arroz Branco Camil Tipo 1 5kg", 24.00, arroz("Camil", 5))
	listing(t, router, "se", "Arroz Branco Sofia Tipo 1 5kg", 20.00, arroz("Sofia", 5))

	c := compareList(t, router, map[string]any{"items": []any{item(camil.CatalogID, 1)}, "allow_equivalents": true})

	line := lineFor(marketByID(t, c, "se"), camil.CatalogID)
	require.False(t, line.IsSubstitute, "se sells the brand that was asked for")
	require.InDelta(t, 24.00, line.Price, 0.001)
}

func TestCompareList_Equivalents_NeverForBrandSensitiveProducts(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	beer := func(brand string) map[string]any {
		return map[string]any{"brand": brand, "generic_key": "cerveja", "size_kind": "volume", "size_amount": 350}
	}
	heineken := listing(t, router, "af", "Cerveja Heineken 350ml", 5.49, beer("Heineken"))
	listing(t, router, "se", "Cerveja Itaipava 350ml", 3.29, beer("Itaipava"))

	c := compareList(t, router, map[string]any{"items": []any{item(heineken.CatalogID, 1)}, "allow_equivalents": true})

	se := marketByID(t, c, "se")
	require.False(t, se.IsComplete, "swapping Heineken for Itaipava is not the platform's call")
	require.Zero(t, se.SubstitutesCount)
}

func TestCompareList_Equivalents_NeverAcrossSizes(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	camil := listing(t, router, "af", "Arroz Branco Camil Tipo 1 5kg", 22.69, arroz("Camil", 5))
	listing(t, router, "se", "Arroz Branco Sofia Tipo 1 2kg", 9.00, arroz("Sofia", 2))

	c := compareList(t, router, map[string]any{"items": []any{item(camil.CatalogID, 1)}, "allow_equivalents": true})

	require.False(t, marketByID(t, c, "se").IsComplete, "2 kg is not a substitute for 5 kg")
}

func TestCompareList_ProductIDResolvesToItsCatalogItem(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	af := listing(t, router, "af", "Café Melitta 500g", 28.90, melitta500())
	listing(t, router, "se", "Café Melitta 500g", 25.90, melitta500())

	c := compareList(t, router, map[string]any{"items": []any{
		map[string]any{"product_id": af.ID, "quantity": 1},
	}})

	require.True(t, marketByID(t, c, "se").IsComplete, "asking with af's listing still finds se's offer")
}

func TestCompareList_RepeatedItemsAreMerged(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	af := listing(t, router, "af", "Café Melitta 500g", 10, melitta500())

	c := compareList(t, router, map[string]any{"items": []any{
		item(af.CatalogID, 1),
		map[string]any{"product_id": af.ID, "quantity": 2},
	}})

	require.Len(t, c.Items, 1)
	require.Equal(t, 3, c.Items[0].Quantity)
	require.InDelta(t, 30.00, marketByID(t, c, "af").TotalCost, 0.001)
}

func TestCompareList_ItemNobodySells_IsUnavailable(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	cafe := listing(t, router, "af", "Café Melitta 500g", 28.90, melitta500())
	rice := listing(t, router, "af", "Arroz Camil 5kg", 22.69, arroz("Camil", 5))
	// nobody sells the rice any more
	_, err := pool.Exec(t.Context(), `DELETE FROM market_products WHERE product_id = $1`, rice.ID)
	require.NoError(t, err)

	c := compareList(t, router, map[string]any{"items": []any{item(cafe.CatalogID, 1), item(rice.CatalogID, 1)}})

	require.True(t, c.Items[0].Available)
	require.False(t, c.Items[1].Available)
	require.Nil(t, c.Items[1].CheapestPrice)
	require.Empty(t, c.BestSingleMarketID, "no market has the whole list")
	require.NotNil(t, c.BestCombination, "the item that can be bought still is")
	require.Nil(t, c.BestCombination.SavingsVsBestSingle)
}

func TestCompareList_Distance_OrdersTiesAndFiltersByRadius(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	// same price at md (0.4 km from the user) and af (1.8 km)
	af := listing(t, router, "af", "Café Melitta 500g", 20, melitta500())
	listing(t, router, "md", "Café Melitta 500g", 20, melitta500())

	near := map[string]any{"items": []any{item(af.CatalogID, 1)}, "lat": -5.0892, "lng": -42.8016}
	c := compareList(t, router, near)
	require.Equal(t, "md", c.Markets[0].Market.ID, "on equal totals the nearer market comes first")
	require.NotNil(t, c.Markets[0].DistanceKm)
	require.InDelta(t, 0.4, *c.Markets[0].DistanceKm, 0.05)

	near["radius_km"] = 1.0
	within := compareList(t, router, near)
	for _, m := range within.Markets {
		require.NotNil(t, m.DistanceKm)
		require.LessOrEqual(t, *m.DistanceKm, 1.0)
	}
	require.Less(t, len(within.Markets), len(c.Markets))

	// without a position there is no distance to show
	nopos := compareList(t, router, map[string]any{"items": []any{item(af.CatalogID, 1)}})
	require.Nil(t, nopos.Markets[0].DistanceKm)
}

func TestCompareList_Validation(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)
	fixtureProduct := "p1" // seed data: has no catalog item

	cases := map[string]map[string]any{
		"no items":           {"items": []any{}},
		"item without id":    {"items": []any{map[string]any{"quantity": 1}}},
		"zero quantity":      {"items": []any{map[string]any{"catalog_id": "x", "quantity": 0}}},
		"lat without lng":    {"items": []any{item("x", 1)}, "lat": -5.0},
		"radius without lat": {"items": []any{item("x", 1)}, "radius_km": 3.0},
		"unknown product":    {"items": []any{map[string]any{"product_id": "nope", "quantity": 1}}},
		"unknown catalog":    {"items": []any{item("nope", 1)}},
		"product not linked": {"items": []any{map[string]any{"product_id": fixtureProduct, "quantity": 1}}},
	}
	for name, body := range cases {
		rec := postJSON(t, router, "/api/comparisons/list", body)
		require.Equal(t, http.StatusBadRequest, rec.Code, name)
	}

	rec := postJSON(t, router, "/api/comparisons/list", cases["product not linked"])
	var out map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	require.Equal(t, fixtureProduct, out["product_id"], "the response says which product couldn't be resolved")
}
