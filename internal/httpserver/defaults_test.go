package httpserver

import "testing"

func TestDeriveInitials(t *testing.T) {
	cases := map[string]string{
		"Atacado Família": "AF",
		"Carrefour":       "CA",
		"A":               "A",
		"":                "??",
	}
	for input, want := range cases {
		if got := deriveInitials(input); got != want {
			t.Errorf("deriveInitials(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestPickPalette_IsDeterministic(t *testing.T) {
	p1, b1 := pickPalette("Mercado do Bairro")
	p2, b2 := pickPalette("Mercado do Bairro")
	if p1 != p2 || b1 != b2 {
		t.Errorf("expected pickPalette to be deterministic for the same name")
	}
}
