package geo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Place is the address of a point, in the same pieces as Address. Any field
// can be empty: OpenStreetMap doesn't know every street or neighbourhood.
type Place struct {
	Street       string
	Neighborhood string
	City         string
	State        string // two-letter code when known ("PI"), else the state's name
	CEP          string
}

// Label is the place as one line: "Rua X, Bairro Y, Teresina - PI".
func (p Place) Label() string {
	var parts []string
	for _, s := range []string{p.Street, p.Neighborhood} {
		if s != "" {
			parts = append(parts, s)
		}
	}
	city := p.City
	if city != "" && p.State != "" {
		city += " - " + p.State
	}
	if city != "" {
		parts = append(parts, city)
	}
	return strings.Join(parts, ", ")
}

// ReverseGeocoder turns coordinates into the address they fall on.
type ReverseGeocoder interface {
	Reverse(ctx context.Context, pt Point) (Place, error)
}

type nominatimReverseResult struct {
	Error   string `json:"error"`
	Address struct {
		Road          string `json:"road"`
		Pedestrian    string `json:"pedestrian"`
		Neighbourhood string `json:"neighbourhood"`
		Suburb        string `json:"suburb"`
		Quarter       string `json:"quarter"`
		City          string `json:"city"`
		Town          string `json:"town"`
		Village       string `json:"village"`
		Municipality  string `json:"municipality"`
		State         string `json:"state"`
		StateCode     string `json:"ISO3166-2-lvl4"`
		Postcode      string `json:"postcode"`
	} `json:"address"`
}

// Reverse looks up the address at pt. It returns ErrNotFound when the point
// isn't on anything Nominatim knows.
func (n *Nominatim) Reverse(ctx context.Context, pt Point) (Place, error) {
	if err := n.wait(ctx); err != nil {
		return Place{}, err
	}

	q := url.Values{}
	q.Set("format", "jsonv2")
	q.Set("lat", strconv.FormatFloat(pt.Lat, 'f', -1, 64))
	q.Set("lon", strconv.FormatFloat(pt.Lng, 'f', -1, 64))
	q.Set("zoom", "18")
	q.Set("addressdetails", "1")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, n.baseURL+"/reverse?"+q.Encode(), nil)
	if err != nil {
		return Place{}, err
	}
	req.Header.Set("User-Agent", n.userAgent)
	req.Header.Set("Accept-Language", "pt-BR")

	resp, err := n.client.Do(req)
	if err != nil {
		return Place{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Place{}, fmt.Errorf("geo: nominatim returned status %d", resp.StatusCode)
	}

	var result nominatimReverseResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return Place{}, fmt.Errorf("geo: decode nominatim response: %w", err)
	}
	if result.Error != "" {
		return Place{}, ErrNotFound
	}

	a := result.Address
	place := Place{
		Street:       firstNonEmpty(a.Road, a.Pedestrian),
		Neighborhood: firstNonEmpty(a.Suburb, a.Neighbourhood, a.Quarter),
		City:         firstNonEmpty(a.City, a.Town, a.Village, a.Municipality),
		State:        firstNonEmpty(strings.TrimPrefix(a.StateCode, "BR-"), a.State),
		CEP:          a.Postcode,
	}
	if place.Label() == "" {
		return Place{}, ErrNotFound
	}
	return place, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
