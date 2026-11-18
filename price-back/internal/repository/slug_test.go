package repository

import (
	"strings"
	"testing"
)

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Mercadão Popular":   "mercadao-popular",
		"  Atacado Família ": "atacado-familia",
		"Café & Cia.":        "cafe-cia",
		"":                   "",
		"###":                "",
	}
	for input, want := range cases {
		if got := slugify(input); got != want {
			t.Errorf("slugify(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestGenerateID_UsesFallbackWhenSlugEmpty(t *testing.T) {
	id := generateID("###", "mercado")
	if !strings.HasPrefix(id, "mercado-") {
		t.Errorf("expected id to start with fallback prefix, got %q", id)
	}
}

func TestGenerateID_IsUnique(t *testing.T) {
	a := generateID("Tomate", "produto")
	b := generateID("Tomate", "produto")
	if a == b {
		t.Errorf("expected two calls to generate distinct ids, both were %q", a)
	}
}
