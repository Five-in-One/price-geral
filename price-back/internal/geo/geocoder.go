package geo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ErrNotFound means the geocoder answered but none of the address
// variations matched a place.
var ErrNotFound = errors.New("geo: address not found")

// Address is the postal address of a market, in the same pieces the
// registration forms collect. Every field is optional.
type Address struct {
	CEP          string
	Street       string
	Number       string
	Neighborhood string
	City         string
	State        string
}

// Empty reports whether there is nothing worth geocoding.
func (a Address) Empty() bool {
	return a.CEP == "" && a.Street == "" && a.Neighborhood == ""
}

// Point is a latitude/longitude pair.
type Point struct {
	Lat float64
	Lng float64
}

// Geocoder turns a postal address into coordinates.
type Geocoder interface {
	Geocode(ctx context.Context, addr Address) (Point, error)
}

// Nominatim geocodes through an OpenStreetMap Nominatim server. The public
// instance (https://nominatim.openstreetmap.org) is free but its usage policy
// requires an identifying User-Agent and at most one request per second, so
// requests are spaced out by minInterval.
type Nominatim struct {
	baseURL     string
	userAgent   string
	client      *http.Client
	minInterval time.Duration

	mu       sync.Mutex
	lastCall time.Time
}

// NewNominatim builds a client for the public Nominatim instance.
func NewNominatim(userAgent string) *Nominatim {
	return &Nominatim{
		baseURL:     "https://nominatim.openstreetmap.org",
		userAgent:   userAgent,
		client:      &http.Client{Timeout: 6 * time.Second},
		minInterval: 1100 * time.Millisecond,
	}
}

type nominatimResult struct {
	Lat string `json:"lat"`
	Lon string `json:"lon"`
}

// Geocode tries progressively looser variations of the address — street with
// number, street alone, postal code, then neighbourhood — and returns the
// first hit. The fallbacks stop at neighbourhood on purpose: a bare city
// would put every market on the city centre and fake a precise distance.
func (n *Nominatim) Geocode(ctx context.Context, addr Address) (Point, error) {
	var lastErr error
	for _, params := range searchVariations(addr) {
		pt, err := n.search(ctx, params)
		if err == nil {
			return pt, nil
		}
		if ctx.Err() != nil {
			return Point{}, ctx.Err()
		}
		if !errors.Is(err, ErrNotFound) {
			lastErr = err
		}
	}
	if lastErr != nil {
		return Point{}, lastErr
	}
	return Point{}, ErrNotFound
}

// searchVariations lists the Nominatim query parameters to try, best first.
func searchVariations(a Address) []url.Values {
	var out []url.Values

	if a.Street != "" && a.City != "" {
		if a.Number != "" {
			out = append(out, url.Values{"street": {a.Number + " " + a.Street}, "city": {a.City}, "state": {a.State}})
		}
		out = append(out, url.Values{"street": {a.Street}, "city": {a.City}, "state": {a.State}})
	}
	if a.CEP != "" {
		out = append(out, url.Values{"postalcode": {a.CEP}})
	}
	if a.Neighborhood != "" && a.City != "" {
		parts := []string{a.Neighborhood, a.City}
		if a.State != "" {
			parts = append(parts, a.State)
		}
		out = append(out, url.Values{"q": {strings.Join(parts, ", ")}})
	}
	return out
}

func (n *Nominatim) search(ctx context.Context, params url.Values) (Point, error) {
	if err := n.wait(ctx); err != nil {
		return Point{}, err
	}

	q := url.Values{}
	for k, v := range params {
		if len(v) > 0 && strings.TrimSpace(v[0]) != "" {
			q.Set(k, v[0])
		}
	}
	q.Set("format", "jsonv2")
	q.Set("limit", "1")
	q.Set("countrycodes", "br")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, n.baseURL+"/search?"+q.Encode(), nil)
	if err != nil {
		return Point{}, err
	}
	req.Header.Set("User-Agent", n.userAgent)
	req.Header.Set("Accept-Language", "pt-BR")

	resp, err := n.client.Do(req)
	if err != nil {
		return Point{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Point{}, fmt.Errorf("geo: nominatim returned status %d", resp.StatusCode)
	}

	var results []nominatimResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return Point{}, fmt.Errorf("geo: decode nominatim response: %w", err)
	}
	if len(results) == 0 {
		return Point{}, ErrNotFound
	}

	lat, err := strconv.ParseFloat(results[0].Lat, 64)
	if err != nil {
		return Point{}, fmt.Errorf("geo: bad latitude %q: %w", results[0].Lat, err)
	}
	lng, err := strconv.ParseFloat(results[0].Lon, 64)
	if err != nil {
		return Point{}, fmt.Errorf("geo: bad longitude %q: %w", results[0].Lon, err)
	}
	return Point{Lat: lat, Lng: lng}, nil
}

// wait blocks until at least minInterval has passed since the previous
// request, honouring Nominatim's one-request-per-second policy.
func (n *Nominatim) wait(ctx context.Context) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if delay := n.minInterval - time.Since(n.lastCall); delay > 0 {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	n.lastCall = time.Now()
	return nil
}
