package repository

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
)

// accentFolds maps common accented Portuguese letters to their plain ASCII
// equivalent, used only for generating readable opaque ids — not a full
// Unicode normalizer.
var accentFolds = strings.NewReplacer(
	"á", "a", "à", "a", "ã", "a", "â", "a", "ä", "a",
	"é", "e", "è", "e", "ê", "e", "ë", "e",
	"í", "i", "ì", "i", "î", "i", "ï", "i",
	"ó", "o", "ò", "o", "õ", "o", "ô", "o", "ö", "o",
	"ú", "u", "ù", "u", "û", "u", "ü", "u",
	"ç", "c", "ñ", "n",
)

// slugify turns a name into a lowercase, hyphenated, ASCII-only string
// suitable for use as part of a generated id.
func slugify(name string) string {
	folded := accentFolds.Replace(strings.ToLower(strings.TrimSpace(name)))

	var b strings.Builder
	lastDash := false
	for _, r := range folded {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}

	return strings.Trim(b.String(), "-")
}

// randomSuffix returns a short random hex string so generated ids don't
// collide across independently-submitted markets/products with similar
// names.
func randomSuffix() string {
	buf := make([]byte, 3)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}

// generateID builds an id like "mercado-bom-preco-a1b2c3" from a name,
// falling back to fallbackPrefix if the name has no usable characters.
func generateID(name, fallbackPrefix string) string {
	slug := slugify(name)
	if slug == "" {
		slug = fallbackPrefix
	}
	return slug + "-" + randomSuffix()
}
