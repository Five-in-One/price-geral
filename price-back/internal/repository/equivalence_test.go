package repository

import "testing"

func TestIsSubstitutable(t *testing.T) {
	cases := map[string]bool{
		"arroz branco tipo1":        true,
		"carioca feijao tipo1":      true,
		"acucar cristal":            true,
		"leite longa integral vida": true,
		"flocao milho":              true,
		"papel higienico":           true,
		"cerveja":                   false,
		"cerveja zero":              false,
		"vinho tinto seco":          false,
		"cafe tradicional vacuo":    false, // not on the list: coffee is brand-loyal
		"chocolate leite":           false, // "leite" must not make chocolate a staple
		"doce leite":                false,
		"":                          false,
	}
	for key, want := range cases {
		if got := IsSubstitutable(key); got != want {
			t.Errorf("IsSubstitutable(%q) = %v, want %v", key, got, want)
		}
	}
}
