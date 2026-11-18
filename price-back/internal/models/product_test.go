package models

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func ptr(v float64) *float64 { return &v }

func TestComputeUnitPrice(t *testing.T) {
	cases := []struct {
		name    string
		product Product
		want    *float64
		per     string
	}{
		{"mass to kg", Product{DefaultPrice: 28.90, SizeKind: "mass", SizeAmount: ptr(500)}, ptr(57.80), "kg"},
		{"volume to L", Product{DefaultPrice: 6.89, SizeKind: "volume", SizeAmount: ptr(1000)}, ptr(6.89), "L"},
		{"length to m", Product{DefaultPrice: 18, SizeKind: "length", SizeAmount: ptr(360)}, ptr(0.05), "m"},
		{"sold by weight", Product{DefaultPrice: 39.90, SoldByWeight: true}, ptr(39.90), "kg"},
		{"count only", Product{DefaultPrice: 60, PackCount: 12}, ptr(5), "un"},
		{"size wins over weight flag", Product{DefaultPrice: 10, SoldByWeight: true, SizeKind: "mass", SizeAmount: ptr(250)}, ptr(40), "kg"},
		{"single unit with no size", Product{DefaultPrice: 5, PackCount: 1}, nil, ""},
		{"nothing known", Product{DefaultPrice: 5}, nil, ""},
		{"unknown kind is ignored", Product{DefaultPrice: 5, SizeKind: "bogus", SizeAmount: ptr(100)}, nil, ""},
	}
	for _, c := range cases {
		p := c.product
		p.ComputeUnitPrice()
		if c.want == nil {
			require.Nil(t, p.UnitPrice, c.name)
			require.Empty(t, p.UnitPriceUnit, c.name)
			continue
		}
		require.NotNil(t, p.UnitPrice, c.name)
		require.InDelta(t, *c.want, *p.UnitPrice, 0.001, c.name)
		require.Equal(t, c.per, p.UnitPriceUnit, c.name)
	}
}
