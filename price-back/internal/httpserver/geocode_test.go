package httpserver_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"priceback/internal/geo"
	"priceback/internal/httpserver"
	"priceback/internal/testutil"
)

// fakeReverseGeocoder returns a fixed place (or error) and remembers the point it was asked about.
type fakeReverseGeocoder struct {
	place geo.Place
	err   error
	got   geo.Point
	calls int
}

func (f *fakeReverseGeocoder) Reverse(_ context.Context, pt geo.Point) (geo.Place, error) {
	f.calls++
	f.got = pt
	return f.place, f.err
}

func getReverse(t *testing.T, reverse geo.ReverseGeocoder, query string) *httptest.ResponseRecorder {
	t.Helper()
	pool := testutil.NewTestPool(t)
	var opts []httpserver.Option
	if reverse != nil {
		opts = append(opts, httpserver.WithReverseGeocoder(reverse))
	}
	router := httpserver.NewRouter(pool, opts...)

	req := httptest.NewRequest(http.MethodGet, "/api/geocode/reverse"+query, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestReverseGeocode_ReturnsAddressAndLabel(t *testing.T) {
	fake := &fakeReverseGeocoder{place: geo.Place{
		Street: "Avenida Homero Castelo Branco", Neighborhood: "Jóquei", City: "Teresina", State: "PI", CEP: "64049-000",
	}}

	rec := getReverse(t, fake, "?lat=-5.0892&lng=-42.8016")

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, geo.Point{Lat: -5.0892, Lng: -42.8016}, fake.got)

	var body map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "Teresina", body["city"])
	require.Equal(t, "PI", body["state"])
	require.Equal(t, "Jóquei", body["neighborhood"])
	require.Equal(t, "Avenida Homero Castelo Branco, Jóquei, Teresina - PI", body["label"])
}

func TestReverseGeocode_InvalidCoordinates_ReturnsBadRequest(t *testing.T) {
	for _, query := range []string{"", "?lat=-5.08", "?lng=-42.8", "?lat=abc&lng=1", "?lat=91&lng=0", "?lat=0&lng=181", "?lat=NaN&lng=0"} {
		fake := &fakeReverseGeocoder{}
		rec := getReverse(t, fake, query)
		require.Equal(t, http.StatusBadRequest, rec.Code, query)
		require.Zero(t, fake.calls, query)
	}
}

func TestReverseGeocode_NothingThere_ReturnsNotFound(t *testing.T) {
	rec := getReverse(t, &fakeReverseGeocoder{err: geo.ErrNotFound}, "?lat=0&lng=0")
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestReverseGeocode_LookupFails_ReturnsBadGateway(t *testing.T) {
	rec := getReverse(t, &fakeReverseGeocoder{err: errors.New("boom")}, "?lat=-5&lng=-42")
	require.Equal(t, http.StatusBadGateway, rec.Code)
}

func TestReverseGeocode_NotConfigured_ReturnsServiceUnavailable(t *testing.T) {
	rec := getReverse(t, nil, "?lat=-5&lng=-42")
	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
}
