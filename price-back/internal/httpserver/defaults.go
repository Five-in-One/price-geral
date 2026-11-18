package httpserver

import (
	"hash/fnv"
	"strings"
)

// deriveInitials picks up to two letters to represent name, mirroring how
// the seeded markets show initials in an avatar (e.g. "Atacado Família" ->
// "AF"). Used to fill in a sensible default when a market is created
// without explicit initials (Partner Company page, bot on-demand
// collection).
func deriveInitials(name string) string {
	words := strings.Fields(name)
	if len(words) == 0 {
		return "??"
	}
	if len(words) == 1 {
		r := []rune(strings.ToUpper(words[0]))
		if len(r) >= 2 {
			return string(r[:2])
		}
		return string(r)
	}
	first := []rune(strings.ToUpper(words[0]))[0]
	second := []rune(strings.ToUpper(words[1]))[0]
	return string([]rune{first, second})
}

// colorPalette holds the same style of primary/background pairs used by
// the seeded markets.
var colorPalette = [][2]string{
	{"#DC2626", "#FDE8E8"},
	{"#0C63E4", "#E5F1FC"},
	{"#7E22CE", "#F3E8FF"},
	{"#1F9E5F", "#E6F5ED"},
	{"#D97706", "#FDF0E2"},
	{"#0891B2", "#E0F7FA"},
	{"#DB2777", "#FCE7F3"},
}

// pickPalette deterministically assigns a color pair based on name, so the
// same market name always gets the same colors.
func pickPalette(name string) (primary, background string) {
	h := fnv.New32a()
	_, _ = h.Write([]byte(name))
	idx := int(h.Sum32() % uint32(len(colorPalette)))
	pair := colorPalette[idx]
	return pair[0], pair[1]
}
