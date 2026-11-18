package repository

import "strings"

// substitutableTypes are the kinds of product where a shopper normally
// accepts another brand when the requested one is missing: staples and
// household basics, bought mostly on price.
//
// It is a whitelist on purpose. The alternative — deciding which products are
// "brand-sensitive" — has to be right for beer, wine, chocolate, coffee,
// snacks, cosmetics and everything else nobody thought of; getting one wrong
// silently swaps something the shopper cares about. With a whitelist the
// worst case is a missed saving.
//
// A word here only says *brand switching is fine* for a product whose generic
// key contains it. What counts as "the same product" is still decided by the
// catalog: same generic key, same package size, same pack count.
var substitutableTypes = map[string]bool{
	"arroz":      true,
	"feijao":     true,
	"acucar":     true,
	"sal":        true,
	"farinha":    true,
	"fuba":       true,
	"flocao":     true,
	"oleo":       true,
	"leite":      true,
	"macarrao":   true,
	"vinagre":    true,
	"agua":       true,
	"papel":      true, // papel higiênico
	"detergente": true,
}

// brandSensitiveWords veto the whitelist. A key is a bag of words, so a word
// like "leite" also shows up in "chocolate ao leite" or "doce de leite" —
// products where the brand is the point.
var brandSensitiveWords = map[string]bool{
	"chocolate": true,
	"doce":      true,
	"biscoito":  true,
	"sorvete":   true,
	"cerveja":   true,
	"vinho":     true,
	"espumante": true,
}

// IsSubstitutable reports whether another brand of the product described by
// genericKey (see products.generic_key: accent-free words, space separated)
// is an acceptable substitute.
func IsSubstitutable(genericKey string) bool {
	allowed := false
	for word := range strings.FieldsSeq(genericKey) {
		if brandSensitiveWords[word] {
			return false
		}
		if substitutableTypes[word] {
			allowed = true
		}
	}
	return allowed
}
