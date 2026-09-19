package httpserver_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"priceback/internal/geo"
	"priceback/internal/httpserver"
	"priceback/internal/models"
	"priceback/internal/testutil"
)

// fakeGeocoder returns a fixed point (or error) and counts its calls.
type fakeGeocoder struct {
	point geo.Point
	err   error
	calls int
}

func (f *fakeGeocoder) Geocode(_ context.Context, _ geo.Address) (geo.Point, error) {
	f.calls++
	return f.point, f.err
}

const (
	userLat = -5.0892
	userLng = -42.8016
)

func listMarketsNear(t *testing.T, router http.Handler, query string) []models.MarketWithDistance {
	t.Helper()
	rec := doRequest(t, router, http.MethodGet, "/api/markets?lat=-5.0892&lng=-42.8016&"+query)
	require.Equal(t, http.StatusOK, rec.Code)
	var out []models.MarketWithDistance
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	return out
}

func findMarket(markets []models.MarketWithDistance, name string) *models.MarketWithDistance {
	for i := range markets {
		if markets[i].Name == name {
			return &markets[i]
		}
	}
	return nil
}

func TestCreateMarket_WithAddress_GeocodesAndHasDistance(t *testing.T) {
	pool := testutil.NewTestPool(t)
	g := &fakeGeocoder{point: geo.Point{Lat: -5.0802, Lng: userLng}}
	router := httpserver.NewRouter(pool, httpserver.WithGeocoder(g))

	rec := postJSON(t, router, "/api/markets", map[string]string{
		"name": "Mercado Geocodado", "street": "Rua A", "city": "Teresina", "state": "PI",
	})
	require.Equal(t, http.StatusCreated, rec.Code)
	require.Equal(t, 1, g.calls)

	m := findMarket(listMarketsNear(t, router, "radius_km=5"), "Mercado Geocodado")
	require.NotNil(t, m)
	require.NotNil(t, m.DistanceKm)
	require.InDelta(t, geo.HaversineKm(userLat, userLng, -5.0802, userLng), *m.DistanceKm, 0.01)
	require.InDelta(t, 1.0, *m.DistanceKm, 0.05)
}

func TestCreateMarket_GeocodingFails_StillCreatedWithoutDistance(t *testing.T) {
	pool := testutil.NewTestPool(t)
	g := &fakeGeocoder{err: geo.ErrNotFound}
	router := httpserver.NewRouter(pool, httpserver.WithGeocoder(g))

	rec := postJSON(t, router, "/api/markets", map[string]string{
		"name": "Mercado Sem Local", "street": "Rua Que Nao Existe", "city": "Teresina",
	})
	require.Equal(t, http.StatusCreated, rec.Code)

	markets := listMarketsNear(t, router, "radius_km=5")
	m := findMarket(markets, "Mercado Sem Local")
	require.NotNil(t, m, "a market with no location must still be listed")
	require.Nil(t, m.DistanceKm, "and must not get an invented distance")

	// Located markets come first, the unlocated one last.
	require.Equal(t, "Mercado Sem Local", markets[len(markets)-1].Name)
}

func TestCreateMarket_ExplicitCoordinates_SkipGeocoder(t *testing.T) {
	pool := testutil.NewTestPool(t)
	g := &fakeGeocoder{err: errors.New("must not be called")}
	router := httpserver.NewRouter(pool, httpserver.WithGeocoder(g))

	rec := postJSON(t, router, "/api/markets", map[string]any{
		"name": "Mercado Coordenado", "street": "Rua B", "city": "Teresina",
		"latitude": -5.0802, "longitude": userLng,
	})
	require.Equal(t, http.StatusCreated, rec.Code)
	require.Zero(t, g.calls)

	m := findMarket(listMarketsNear(t, router, "radius_km=5"), "Mercado Coordenado")
	require.NotNil(t, m)
	require.NotNil(t, m.DistanceKm)
}

func TestCreateMarket_NoAddress_DoesNotCallGeocoder(t *testing.T) {
	pool := testutil.NewTestPool(t)
	g := &fakeGeocoder{point: geo.Point{Lat: -5.0802, Lng: userLng}}
	router := httpserver.NewRouter(pool, httpserver.WithGeocoder(g))

	rec := postJSON(t, router, "/api/markets", map[string]string{"name": "Mercado Sem Endereco"})
	require.Equal(t, http.StatusCreated, rec.Code)
	require.Zero(t, g.calls)

	m := findMarket(listMarketsNear(t, router, "radius_km=5"), "Mercado Sem Endereco")
	require.NotNil(t, m)
	require.Nil(t, m.DistanceKm)
}

func TestCreateMarket_NoGeocoderConfigured_HasNoDistance(t *testing.T) {
	pool := testutil.NewTestPool(t)
	router := httpserver.NewRouter(pool)

	rec := postJSON(t, router, "/api/markets", map[string]string{
		"name": "Mercado Offline", "street": "Rua C", "city": "Teresina",
	})
	require.Equal(t, http.StatusCreated, rec.Code)

	m := findMarket(listMarketsNear(t, router, "radius_km=5"), "Mercado Offline")
	require.NotNil(t, m)
	require.Nil(t, m.DistanceKm)
}

func TestCreateMarket_ResubmitWithFailedGeocoding_KeepsRealCoordinates(t *testing.T) {
	pool := testutil.NewTestPool(t)
	g := &fakeGeocoder{point: geo.Point{Lat: -5.0802, Lng: userLng}}
	router := httpserver.NewRouter(pool, httpserver.WithGeocoder(g))

	body := map[string]string{"name": "Mercado Persistente", "street": "Rua D", "city": "Teresina"}
	require.Equal(t, http.StatusCreated, postJSON(t, router, "/api/markets", body).Code)

	g.err = geo.ErrNotFound // e.g. Nominatim is down on the second submit
	require.Equal(t, http.StatusCreated, postJSON(t, router, "/api/markets", body).Code)

	m := findMarket(listMarketsNear(t, router, "radius_km=5"), "Mercado Persistente")
	require.NotNil(t, m)
	require.NotNil(t, m.DistanceKm, "a failed re-geocode must not wipe the stored location")
	require.InDelta(t, 1.0, *m.DistanceKm, 0.05)
}

func TestCreateMarket_ResubmitWithAddress_FillsMissingLocation(t *testing.T) {
	pool := testutil.NewTestPool(t)
	g := &fakeGeocoder{err: geo.ErrNotFound}
	router := httpserver.NewRouter(pool, httpserver.WithGeocoder(g))

	// First registered without a usable location...
	require.Equal(t, http.StatusCreated, postJSON(t, router, "/api/markets", map[string]string{"name": "Mercado Tardio"}).Code)
	require.Nil(t, findMarket(listMarketsNear(t, router, "radius_km=5"), "Mercado Tardio").DistanceKm)

	// ...then the admin resubmits it with an address that geocodes.
	g.err = nil
	g.point = geo.Point{Lat: -5.0802, Lng: userLng}
	require.Equal(t, http.StatusCreated, postJSON(t, router, "/api/markets", map[string]string{
		"name": "Mercado Tardio", "street": "Rua E", "city": "Teresina",
	}).Code)

	m := findMarket(listMarketsNear(t, router, "radius_km=5"), "Mercado Tardio")
	require.NotNil(t, m.DistanceKm)
	require.InDelta(t, 1.0, *m.DistanceKm, 0.05)
}

func TestUpdateMarket_ChangedAddress_IsStoredAndRegeocoded(t *testing.T) {
	pool := testutil.NewTestPool(t)
	g := &fakeGeocoder{point: geo.Point{Lat: -5.0802, Lng: userLng}}
	router := httpserver.NewRouter(pool, httpserver.WithGeocoder(g))

	rec := patchJSON(t, router, "/api/markets/af", map[string]any{
		"cep": "64000-000", "street": "Rua Nova", "number": "77", "neighborhood": "Centro", "city": "Teresina", "state": "PI",
	})
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 1, g.calls)

	var market models.Market
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &market))
	require.Equal(t, "Rua Nova", market.Street)
	require.Equal(t, "77", market.Number)
	require.Equal(t, "Teresina", market.City)
	require.True(t, market.HasLocation)

	// The fixture had "af" at -5.072984; it must now sit at the geocoded point.
	m := findMarket(listMarketsNear(t, router, "all=true"), "Atacado Família")
	require.NotNil(t, m.DistanceKm)
	require.InDelta(t, geo.HaversineKm(userLat, userLng, -5.0802, userLng), *m.DistanceKm, 0.01)
}

func TestUpdateMarket_AddressGeocodingFails_DropsLocation(t *testing.T) {
	pool := testutil.NewTestPool(t)
	g := &fakeGeocoder{err: geo.ErrNotFound}
	router := httpserver.NewRouter(pool, httpserver.WithGeocoder(g))

	rec := patchJSON(t, router, "/api/markets/af", map[string]any{"street": "Rua Que Nao Existe", "city": "Teresina"})
	require.Equal(t, http.StatusOK, rec.Code)

	var market models.Market
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &market))
	require.Equal(t, "Rua Que Nao Existe", market.Street, "the address is still saved")
	require.False(t, market.HasLocation, "but the old coordinates no longer match it")

	m := findMarket(listMarketsNear(t, router, "all=true"), "Atacado Família")
	require.Nil(t, m.DistanceKm)
}

func TestUpdateMarket_NameOnly_DoesNotGeocode(t *testing.T) {
	pool := testutil.NewTestPool(t)
	g := &fakeGeocoder{point: geo.Point{Lat: 1, Lng: 1}}
	router := httpserver.NewRouter(pool, httpserver.WithGeocoder(g))

	rec := patchJSON(t, router, "/api/markets/af", map[string]any{"name": "Atacado Família Ltda"})
	require.Equal(t, http.StatusOK, rec.Code)
	require.Zero(t, g.calls)

	m := findMarket(listMarketsNear(t, router, "all=true"), "Atacado Família Ltda")
	require.NotNil(t, m.DistanceKm, "location kept when the address wasn't touched")
}

func TestUpdateMarket_SameAddressResent_DoesNotGeocode(t *testing.T) {
	pool := testutil.NewTestPool(t)
	g := &fakeGeocoder{point: geo.Point{Lat: -5.0802, Lng: userLng}}
	router := httpserver.NewRouter(pool, httpserver.WithGeocoder(g))

	body := map[string]any{"street": "Rua Fixa", "city": "Teresina"}
	require.Equal(t, http.StatusOK, patchJSON(t, router, "/api/markets/af", body).Code)
	require.Equal(t, 1, g.calls)

	// The admin modal always sends the whole address; an unchanged one must
	// not cost another Nominatim call.
	require.Equal(t, http.StatusOK, patchJSON(t, router, "/api/markets/af", body).Code)
	require.Equal(t, 1, g.calls)
}

func TestUpdateMarket_ClearingAddress_DropsLocation(t *testing.T) {
	pool := testutil.NewTestPool(t)
	g := &fakeGeocoder{point: geo.Point{Lat: -5.0802, Lng: userLng}}
	router := httpserver.NewRouter(pool, httpserver.WithGeocoder(g))

	require.Equal(t, http.StatusOK, patchJSON(t, router, "/api/markets/af", map[string]any{"street": "Rua X", "city": "Teresina"}).Code)
	rec := patchJSON(t, router, "/api/markets/af", map[string]any{"street": "", "city": ""})
	require.Equal(t, http.StatusOK, rec.Code)

	var market models.Market
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &market))
	require.Empty(t, market.Street)
	require.False(t, market.HasLocation)
}
