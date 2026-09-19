package geo

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func newTestNominatim(baseURL string) *Nominatim {
	n := NewNominatim("PriceApp-test")
	n.baseURL = baseURL
	n.minInterval = 0
	return n
}

func TestNominatim_FirstVariationHits_ReturnsPointAndSendsUserAgent(t *testing.T) {
	var gotUA, gotStreet, gotCity string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		gotStreet = r.URL.Query().Get("street")
		gotCity = r.URL.Query().Get("city")
		_, _ = w.Write([]byte(`[{"lat":"-5.0802","lon":"-42.8100"}]`))
	}))
	defer srv.Close()

	pt, err := newTestNominatim(srv.URL).Geocode(context.Background(), Address{
		Street: "Avenida Homero Castelo Branco", Number: "1500", City: "Teresina", State: "PI",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pt.Lat != -5.0802 || pt.Lng != -42.8100 {
		t.Fatalf("unexpected point: %+v", pt)
	}
	if gotUA != "PriceApp-test" {
		t.Fatalf("User-Agent not sent, got %q", gotUA)
	}
	if gotStreet != "1500 Avenida Homero Castelo Branco" || gotCity != "Teresina" {
		t.Fatalf("expected number+street and city in query, got street=%q city=%q", gotStreet, gotCity)
	}
}

func TestNominatim_FallsBackWhenMoreSpecificQueryMisses(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Miss the "number + street" query, hit the plain street one.
		if atomic.AddInt32(&calls, 1) == 1 {
			_, _ = w.Write([]byte(`[]`))
			return
		}
		_, _ = w.Write([]byte(`[{"lat":"-5.1","lon":"-42.8"}]`))
	}))
	defer srv.Close()

	pt, err := newTestNominatim(srv.URL).Geocode(context.Background(), Address{
		Street: "Rua das Flores", Number: "10", City: "Teresina", State: "PI",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pt.Lat != -5.1 || atomic.LoadInt32(&calls) != 2 {
		t.Fatalf("expected a hit on the 2nd query, got point %+v after %d calls", pt, calls)
	}
}

func TestNominatim_NothingMatches_ReturnsErrNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	_, err := newTestNominatim(srv.URL).Geocode(context.Background(), Address{
		Street: "Rua Inexistente", City: "Teresina", CEP: "64000-000", Neighborhood: "Centro",
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestNominatim_ServerError_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	_, err := newTestNominatim(srv.URL).Geocode(context.Background(), Address{CEP: "64000-000"})
	if err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("expected a non-NotFound error, got %v", err)
	}
}

func TestNominatim_BareCity_IsNotGeocoded(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		_, _ = w.Write([]byte(`[{"lat":"-5.0","lon":"-42.0"}]`))
	}))
	defer srv.Close()

	_, err := newTestNominatim(srv.URL).Geocode(context.Background(), Address{City: "Teresina", State: "PI"})
	if !errors.Is(err, ErrNotFound) || atomic.LoadInt32(&calls) != 0 {
		t.Fatalf("a city alone must not hit the network (would pin every market to the centre); err=%v calls=%d", err, calls)
	}
}

func TestNominatim_RespectsMinInterval(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"lat":"-5.0","lon":"-42.0"}]`))
	}))
	defer srv.Close()

	n := newTestNominatim(srv.URL)
	n.minInterval = 120 * time.Millisecond

	start := time.Now()
	for range 2 {
		if _, err := n.Geocode(context.Background(), Address{CEP: "64000-000"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	// First call is immediate, the second must wait out the interval.
	if elapsed := time.Since(start); elapsed < 100*time.Millisecond {
		t.Fatalf("requests were not spaced out, took only %v", elapsed)
	}
}

func TestAddress_Empty(t *testing.T) {
	if !(Address{City: "Teresina", State: "PI"}).Empty() {
		t.Fatal("city+state alone should count as empty")
	}
	if (Address{CEP: "64000-000"}).Empty() {
		t.Fatal("a CEP is enough to try geocoding")
	}
}

func TestNominatim_Reverse_MapsAddressPieces(t *testing.T) {
	var gotPath, gotLat, gotLon, gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotLat, gotLon, gotUA = r.URL.Path, r.URL.Query().Get("lat"), r.URL.Query().Get("lon"), r.Header.Get("User-Agent")
		_, _ = w.Write([]byte(`{"address":{"road":"Avenida Homero Castelo Branco","suburb":"Jóquei","city":"Teresina","state":"Piauí","ISO3166-2-lvl4":"BR-PI","postcode":"64049-000"}}`))
	}))
	defer srv.Close()

	place, err := newTestNominatim(srv.URL).Reverse(context.Background(), Point{Lat: -5.0802, Lng: -42.81})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != "/reverse" || gotLat != "-5.0802" || gotLon != "-42.81" || gotUA != "PriceApp-test" {
		t.Fatalf("unexpected request: path=%q lat=%q lon=%q ua=%q", gotPath, gotLat, gotLon, gotUA)
	}
	want := Place{Street: "Avenida Homero Castelo Branco", Neighborhood: "Jóquei", City: "Teresina", State: "PI", CEP: "64049-000"}
	if place != want {
		t.Fatalf("got %+v, want %+v", place, want)
	}
	if got := place.Label(); got != "Avenida Homero Castelo Branco, Jóquei, Teresina - PI" {
		t.Fatalf("unexpected label %q", got)
	}
}

func TestNominatim_Reverse_FallsBackToTownAndStateName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"address":{"town":"Altos","state":"Piauí"}}`))
	}))
	defer srv.Close()

	place, err := newTestNominatim(srv.URL).Reverse(context.Background(), Point{Lat: -5.03, Lng: -42.46})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := place.Label(); got != "Altos - Piauí" {
		t.Fatalf("unexpected label %q", got)
	}
}

func TestNominatim_Reverse_NothingThere_ReturnsNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"error":"Unable to geocode"}`))
	}))
	defer srv.Close()

	_, err := newTestNominatim(srv.URL).Reverse(context.Background(), Point{Lat: 0, Lng: 0})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestNominatim_Reverse_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	_, err := newTestNominatim(srv.URL).Reverse(context.Background(), Point{Lat: -5, Lng: -42})
	if err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("expected a non-NotFound error, got %v", err)
	}
}
