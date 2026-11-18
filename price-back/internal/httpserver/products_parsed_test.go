package httpserver_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"priceback/internal/httpserver"
	"priceback/internal/models"
	"priceback/internal/testutil"
)

func decodeProduct(t *testing.T, body []byte) models.Product {
	t.Helper()
	var p models.Product
	require.NoError(t, json.Unmarshal(body, &p))
	return p
}

func TestCreateProduct_WithParsedFields_StoresAndReturnsThemWithUnitPrice(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	rec := postJSON(t, router, "/api/products", map[string]any{
		"name": "Café Melitta Tradicional Vácuo 500g", "unit": "un", "category": "GRÃOS",
		"price": 28.90, "market_id": "af", "source": "bot",
		"brand": "Melitta", "generic_key": "cafe tradicional vacuo",
		"pack_count": 1, "size_kind": "mass", "size_amount": 500, "size_label": "500 g",
	})
	require.Equal(t, http.StatusCreated, rec.Code)

	p := decodeProduct(t, rec.Body.Bytes())
	require.Equal(t, "Melitta", p.Brand)
	require.Equal(t, "cafe tradicional vacuo", p.GenericKey)
	require.Equal(t, "mass", p.SizeKind)
	require.NotNil(t, p.SizeAmount)
	require.InDelta(t, 500, *p.SizeAmount, 0.001)
	require.Equal(t, "500 g", p.SizeLabel)
	require.NotNil(t, p.UnitPrice)
	require.InDelta(t, 57.80, *p.UnitPrice, 0.001)
	require.Equal(t, "kg", p.UnitPriceUnit)
}

func TestCreateProduct_UnitPrice_ByKind(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	cases := []struct {
		name  string
		extra map[string]any
		price float64
		want  float64
		per   string
	}{
		{"Leite 1L", map[string]any{"size_kind": "volume", "size_amount": 1000}, 6.89, 6.89, "L"},
		{"Papel 12x30m", map[string]any{"size_kind": "length", "size_amount": 360, "pack_count": 12}, 18.00, 0.05, "m"},
		{"Queijo KG", map[string]any{"sold_by_weight": true}, 39.90, 39.90, "kg"},
		{"Fralda C12", map[string]any{"pack_count": 12}, 60.00, 5.00, "un"},
	}
	for _, c := range cases {
		body := map[string]any{
			"name": c.name, "unit": "un", "category": "OUTROS", "price": c.price, "market_id": "af", "source": "bot",
		}
		for k, v := range c.extra {
			body[k] = v
		}
		rec := postJSON(t, router, "/api/products", body)
		require.Equal(t, http.StatusCreated, rec.Code, c.name)

		p := decodeProduct(t, rec.Body.Bytes())
		require.NotNil(t, p.UnitPrice, c.name)
		require.InDelta(t, c.want, *p.UnitPrice, 0.001, c.name)
		require.Equal(t, c.per, p.UnitPriceUnit, c.name)
	}
}

func TestCreateProduct_WithoutParsedFields_HasNoUnitPrice(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	rec := postJSON(t, router, "/api/products", map[string]any{
		"name": "Produto Sem Tamanho", "unit": "un", "category": "OUTROS", "price": 5, "market_id": "af",
	})
	require.Equal(t, http.StatusCreated, rec.Code)

	p := decodeProduct(t, rec.Body.Bytes())
	require.Nil(t, p.UnitPrice)
	require.Nil(t, p.SizeAmount)
	require.Equal(t, 1, p.PackCount)
	require.Empty(t, p.Brand)
}

func TestCreateProduct_ResubmitWithoutParsedFields_KeepsThem(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	full := map[string]any{
		"name": "Arroz Camil 5kg", "unit": "kg", "category": "GRÃOS", "price": 22.69, "market_id": "af",
		"source": "bot", "brand": "Camil", "size_kind": "mass", "size_amount": 5000, "size_label": "5 kg",
	}
	require.Equal(t, http.StatusCreated, postJSON(t, router, "/api/products", full).Code)

	// e.g. the partner page resubmitting the same product with a new price
	rec := postJSON(t, router, "/api/products", map[string]any{
		"name": "Arroz Camil 5kg", "unit": "kg", "category": "GRÃOS", "price": 23.50, "market_id": "af",
	})
	require.Equal(t, http.StatusCreated, rec.Code)

	p := decodeProduct(t, rec.Body.Bytes())
	require.InDelta(t, 23.50, p.DefaultPrice, 0.001)
	require.Equal(t, "Camil", p.Brand, "parsed fields survive a resubmission that doesn't carry them")
	require.NotNil(t, p.SizeAmount)
}

func TestCreateProduct_CollectedAt_IsStoredOrDefaultsToNow(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	seen := time.Date(2026, 9, 1, 10, 30, 0, 0, time.UTC)
	rec := postJSON(t, router, "/api/products", map[string]any{
		"name": "Com Data", "unit": "un", "category": "OUTROS", "price": 5, "market_id": "af",
		"collected_at": seen.Format(time.RFC3339),
	})
	require.Equal(t, http.StatusCreated, rec.Code)
	require.True(t, decodeProduct(t, rec.Body.Bytes()).CollectedAt.Equal(seen))

	rec = postJSON(t, router, "/api/products", map[string]any{
		"name": "Sem Data", "unit": "un", "category": "OUTROS", "price": 5, "market_id": "af",
	})
	require.Equal(t, http.StatusCreated, rec.Code)
	require.WithinDuration(t, time.Now(), decodeProduct(t, rec.Body.Bytes()).CollectedAt, time.Minute)
}

func TestCreateProduct_InvalidParsedFields_ReturnBadRequest(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	base := func(extra map[string]any) map[string]any {
		body := map[string]any{"name": "X", "unit": "un", "category": "OUTROS", "price": 5, "market_id": "af"}
		for k, v := range extra {
			body[k] = v
		}
		return body
	}

	for name, extra := range map[string]map[string]any{
		"kind without amount": {"size_kind": "mass"},
		"amount without kind": {"size_amount": 500},
		"unknown kind":        {"size_kind": "weight", "size_amount": 500},
		"zero amount":         {"size_kind": "mass", "size_amount": 0},
		"negative pack count": {"pack_count": -2},
		"collected in future": {"collected_at": time.Now().Add(48 * time.Hour).Format(time.RFC3339)},
	} {
		rec := postJSON(t, router, "/api/products", base(extra))
		require.Equal(t, http.StatusBadRequest, rec.Code, name)
	}
}

func TestListProducts_ReturnsParsedFieldsAndUnitPrice(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	require.Equal(t, http.StatusCreated, postJSON(t, router, "/api/products", map[string]any{
		"name": "Feijão Kicaldo 1kg", "unit": "kg", "category": "GRÃOS", "price": 7.49, "market_id": "af",
		"brand": "Kicaldo", "size_kind": "mass", "size_amount": 1000,
	}).Code)

	rec := doRequest(t, router, http.MethodGet, "/api/products?search=kicaldo")
	require.Equal(t, http.StatusOK, rec.Code)
	var list []models.Product
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
	require.Len(t, list, 1)
	require.Equal(t, "Kicaldo", list[0].Brand)
	require.NotNil(t, list[0].UnitPrice)
	require.InDelta(t, 7.49, *list[0].UnitPrice, 0.001)
}

func TestUpdateProduct_BackfillsParsedFieldsWithoutTouchingCollectedAt(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	seen := time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC)
	created := postJSON(t, router, "/api/products", map[string]any{
		"name": "Açúcar Caravelas 1kg", "unit": "kg", "category": "GRÃOS", "price": 2.99, "market_id": "af",
		"collected_at": seen.Format(time.RFC3339),
	})
	require.Equal(t, http.StatusCreated, created.Code)
	id := decodeProduct(t, created.Body.Bytes()).ID

	rec := patchJSON(t, router, "/api/products/"+id, map[string]any{
		"brand": "Caravelas", "generic_key": "acucar refinado", "size_kind": "mass",
		"size_amount": 1000, "size_label": "1 kg", "pack_count": 1,
	})
	require.Equal(t, http.StatusOK, rec.Code)

	p := decodeProduct(t, rec.Body.Bytes())
	require.Equal(t, "Caravelas", p.Brand)
	require.NotNil(t, p.UnitPrice)
	require.InDelta(t, 2.99, *p.UnitPrice, 0.001)
	require.True(t, p.CollectedAt.Equal(seen), "a backfill must not make an old price look freshly collected")
}

func TestUpdateProduct_NewPriceRefreshesCollectedAt(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	seen := time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC)
	created := postJSON(t, router, "/api/products", map[string]any{
		"name": "Preço Velho", "unit": "un", "category": "OUTROS", "price": 5, "market_id": "af",
		"collected_at": seen.Format(time.RFC3339),
	})
	id := decodeProduct(t, created.Body.Bytes()).ID

	rec := patchJSON(t, router, "/api/products/"+id, map[string]any{"price": 6.5})
	require.Equal(t, http.StatusOK, rec.Code)
	require.WithinDuration(t, time.Now(), decodeProduct(t, rec.Body.Bytes()).CollectedAt, time.Minute)
}

func TestUpdateProduct_ClearingSize(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	created := postJSON(t, router, "/api/products", map[string]any{
		"name": "Tinha Tamanho", "unit": "un", "category": "OUTROS", "price": 5, "market_id": "af",
		"size_kind": "mass", "size_amount": 500,
	})
	id := decodeProduct(t, created.Body.Bytes()).ID

	rec := patchJSON(t, router, "/api/products/"+id, map[string]any{"size_kind": ""})
	require.Equal(t, http.StatusOK, rec.Code)
	p := decodeProduct(t, rec.Body.Bytes())
	require.Nil(t, p.SizeAmount)
	require.Empty(t, p.SizeKind)
	require.Nil(t, p.UnitPrice)
}

func TestUpdateProduct_InvalidParsedFields_ReturnBadRequest(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	for name, body := range map[string]map[string]any{
		"amount without kind": {"size_amount": 500},
		"kind without amount": {"size_kind": "mass"},
		"unknown kind":        {"size_kind": "weight", "size_amount": 5},
		"pack count zero":     {"pack_count": 0},
	} {
		rec := patchJSON(t, router, "/api/products/p7", body)
		require.Equal(t, http.StatusBadRequest, rec.Code, name)
	}
}
