package geo

import (
	"math"
	"testing"
)

func TestHaversineKm_SamePoint(t *testing.T) {
	d := HaversineKm(-5.0892, -42.8016, -5.0892, -42.8016)
	if math.Abs(d) > 1e-9 {
		t.Fatalf("expected 0km for identical points, got %f", d)
	}
}

func TestHaversineKm_KnownDistance(t *testing.T) {
	// ~0.4km north offset used to seed the "Mercado do Bairro" market.
	d := HaversineKm(-5.0892, -42.8016, -5.085596, -42.8016)
	if math.Abs(d-0.4) > 0.01 {
		t.Fatalf("expected ~0.4km, got %f", d)
	}
}
