package repository

import (
	"regexp"
	"strings"
)

// Staples: what a basic product *is*, read from its name, so two brands of the
// same thing can stand in for each other in a list comparison.
//
// The generic key alone is too strict for that: it keeps every word the parser
// didn't recognise as a brand ("oleo sinha soja" vs "oleo soja"), listing
// noise ("leite longa vida com tampa integral" vs "leite integral") and misses
// words the name leaves implicit ("arroz tipo1" is white rice). Here a staple
// is a product type plus the few qualifiers that really change the product
// (integral × desnatado, tipo 1 × tipo 2, carioca × preto); everything else
// in the name, brand included, is ignored.
//
// It is a whitelist on purpose, like substitutableTypes: a product that
// matches no type is never swapped for another brand.

// staple is a classified product: its type and the value of each qualifier.
type staple struct {
	Type       string
	Qualifiers string // "group=value;group=value", in the type's group order
}

// qualifier is one property that tells products of a type apart. The first
// option whose phrase is in the name wins; with none, the value is def. An
// empty def means the name must say it: without it the product isn't
// classified (it could be any of them).
type qualifier struct {
	group   string
	options []option
	def     string
}

type option struct {
	phrase string // words as they appear in the normalised name
	value  string
}

type stapleType struct {
	name string
	// starts: the name must begin with one of these phrases ("leite", "oleo
	// de soja"…); product names put the type first.
	starts []string
	// excludes: any of these phrases means it is something else ("leite de
	// coco", "sal para churrasco").
	excludes   []string
	qualifiers []qualifier
}

func opts(pairs ...string) []option {
	out := make([]option, 0, len(pairs)/2)
	for i := 0; i+1 < len(pairs); i += 2 {
		out = append(out, option{phrase: pairs[i], value: pairs[i+1]})
	}
	return out
}

var lactose = qualifier{group: "lactose", options: opts("zero lactose", "zero", "sem lactose", "zero"), def: "normal"}

// stapleTypes: checked in order, so the more specific types come first
// ("leite condensado" before "leite").
var stapleTypes = []stapleType{
	{
		name:     "arroz",
		starts:   []string{"arroz"},
		excludes: []string{"doce", "tempero", "biscoito", "cookies", "farinha", "flocao", "flocos", "risoto", "pronto", "bolinho"},
		qualifiers: []qualifier{
			{group: "tipo", options: opts("parboilizado", "parboilizado", "integral", "integral", "arboreo", "arboreo",
				"japones", "japones", "cateto", "cateto", "agulhinha", "branco", "branco", "branco"), def: "branco"},
			{group: "classe", options: opts("tipo1", "1", "tipo2", "2", "tipo3", "3"), def: "1"},
		},
	},
	{
		name:     "feijao",
		starts:   []string{"feijao"},
		excludes: []string{"tempero", "caldo", "pronto", "feijoada", "sopa", "cozido", "lata"},
		qualifiers: []qualifier{
			// fradinho before branco: "Feijao Branco Fradinho" is fradinho
			{group: "variedade", options: opts("fradinho", "fradinho", "carioca", "carioca", "preto", "preto",
				"sempre verde", "sempre verde", "de corda", "corda", "corda", "corda", "macassar", "corda",
				"rajado", "rajado", "jalo", "jalo", "bolinha", "bolinha", "quebra cadeira", "quebra cadeira",
				"vermelho", "vermelho", "branco", "branco")},
			{group: "classe", options: opts("tipo1", "1", "tipo2", "2", "tipo3", "3"), def: "1"},
		},
	},
	{
		name:     "acucar",
		starts:   []string{"acucar"},
		excludes: []string{"zero", "diet", "light", "biscoito", "granola"},
		qualifiers: []qualifier{
			{group: "tipo", options: opts("demerara", "demerara", "mascavo", "mascavo", "refinado", "refinado",
				"confeiteiro", "confeiteiro", "cristal", "cristal", "organico", "organico"), def: "cristal"},
		},
	},
	{
		name:     "sal",
		starts:   []string{"sal"},
		excludes: []string{"churrasco", "temperado", "parrilla", "parrila", "chimichurri", "chimichuri", "grill", "ervas", "light", "tempero"},
		qualifiers: []qualifier{
			{group: "tipo", options: opts("grosso", "grosso", "marinho", "marinho", "rosa", "rosa", "refinado", "refinado"), def: "refinado"},
		},
	},
	{
		name:     "oleo",
		starts:   []string{"oleo"},
		excludes: []string{"amendoas", "composto", "corporal", "cabelo", "spray"},
		qualifiers: []qualifier{
			{group: "tipo", options: opts("soja", "soja", "canola", "canola", "girassol", "girassol", "milho", "milho", "algodao", "algodao")},
		},
	},
	{
		name:     "leite condensado",
		starts:   []string{"leite condensado", "mistura lactea condensada"},
		excludes: []string{"diet", "cozido", "doce"},
		qualifiers: []qualifier{
			{group: "gordura", options: opts("semidesnatado", "semidesnatado", "desnatado", "desnatado", "light", "semidesnatado"), def: "integral"},
			lactose,
		},
	},
	{
		name:     "creme de leite",
		starts:   []string{"creme de leite"},
		excludes: []string{"culinario"},
		qualifiers: []qualifier{
			{group: "gordura", options: opts("leve", "leve", "light", "leve"), def: "normal"},
			lactose,
		},
	},
	{
		name:     "leite",
		starts:   []string{"leite"},
		excludes: []string{"po", "condensado", "coco", "fermentado", "doce", "achocolatado", "chocolate", "cabra", "aveia", "amendoas", "soja", "vegetal", "creme", "mistura"},
		qualifiers: []qualifier{
			{group: "gordura", options: opts("semidesnatado", "semidesnatado", "desnatado", "desnatado", "integral", "integral"), def: "integral"},
			lactose,
		},
	},
	{
		name:     "farinha de trigo",
		starts:   []string{"farinha de trigo", "farinha trigo"},
		excludes: []string{"pre mistura", "bolo"},
		qualifiers: []qualifier{
			{group: "fermento", options: opts("com fermento", "com", "sem fermento", "sem"), def: "sem"},
			{group: "tipo", options: opts("integral", "integral"), def: "comum"},
		},
	},
	{
		name:   "farinha de mandioca",
		starts: []string{"farinha de mandioca", "farinha mandioca"},
		qualifiers: []qualifier{
			{group: "cor", options: opts("amarela", "amarela", "branca", "branca"), def: "branca"},
			{group: "textura", options: opts("fina", "fina", "grossa", "grossa", "torrada", "torrada"), def: "comum"},
		},
	},
	{
		name:     "flocao de milho",
		starts:   []string{"flocao", "farinha de milho flocada", "farinha milho flocada"},
		excludes: []string{"arroz"},
	},
	{
		name:     "fuba",
		starts:   []string{"fuba"},
		excludes: []string{"bolo"},
	},
	{
		name:     "macarrao",
		starts:   []string{"macarrao", "massa"},
		excludes: []string{"instantaneo", "lamen", "cup", "pronto", "salada"},
		qualifiers: []qualifier{
			{group: "formato", options: opts("espaguete", "espaguete", "spaghetti", "espaguete", "parafuso", "parafuso",
				"fusilli", "parafuso", "penne", "penne", "talharim", "talharim", "padre nosso", "padre nosso",
				"ave maria", "ave maria", "cabelo de anjo", "cabelo de anjo", "letrinha", "letrinha",
				"conchinha", "conchinha", "argolinha", "argolinha", "lasanha", "lasanha", "farfalle", "farfalle",
				"gravata", "farfalle", "pena", "penne", "caracol", "caracol")},
			{group: "massa", options: opts("ovos", "ovos", "semola", "semola", "semolina", "semola", "integral", "integral"), def: "comum"},
		},
	},
	{
		name:     "molho de tomate",
		starts:   []string{"molho de tomate", "molho tomate"},
		excludes: []string{"sardinha", "atum"},
		qualifiers: []qualifier{
			{group: "sabor", options: opts("manjericao", "manjericao", "pizza", "pizza", "bolonhesa", "bolonhesa",
				"ervas", "ervas", "azeitona", "azeitona", "cebola", "cebola", "picante", "picante",
				"light", "light", "caseiro", "caseiro"), def: "tradicional"},
		},
	},
	{
		name:   "extrato de tomate",
		starts: []string{"extrato de tomate", "extrato tomate"},
	},
	{
		name:     "vinagre",
		starts:   []string{"vinagre"},
		excludes: []string{"aromatizado", "temperado", "balsamico"},
		qualifiers: []qualifier{
			{group: "tipo", options: opts("vinho tinto", "vinho tinto", "vinho branco", "vinho branco", "maca", "maca",
				"fruta", "maca", "arroz", "arroz", "alcool", "alcool")},
		},
	},
	{
		name:   "agua mineral",
		starts: []string{"agua mineral", "agua"},
		excludes: []string{"coco", "tonica", "sanitaria", "saborizada", "sabor", "oxigenada", "micelar", "perfume",
			"h2o", "h2oh", "limao", "laranja", "uva", "maracuja", "citrus", "limonada"},
		qualifiers: []qualifier{
			{group: "gas", options: opts("com gas", "com gas", "gaseificada", "com gas", "sem gas", "sem gas"), def: "sem gas"},
		},
	},
	{
		name:   "papel higienico",
		starts: []string{"papel higienico", "papel hig"},
		qualifiers: []qualifier{
			{group: "folha", options: opts("folha tripla", "tripla", "folha dupla", "dupla", "folha simples", "simples"), def: "simples"},
		},
	},
	{
		name:     "detergente",
		starts:   []string{"detergente"},
		excludes: []string{"po", "maquina", "lava roupas", "roupa"},
	},
}

var (
	typeWordRe = regexp.MustCompile(`\b(?:tipo|tp|t)\s*-?\s*([123])\b`)
	nonWordRe  = regexp.MustCompile(`[^a-z0-9]+`)
	accents    = strings.NewReplacer(
		"á", "a", "à", "a", "â", "a", "ã", "a", "ä", "a", "é", "e", "ê", "e", "è", "e", "í", "i", "î", "i",
		"ó", "o", "ô", "o", "õ", "o", "ö", "o", "ú", "u", "ü", "u", "ç", "c", "ñ", "n",
	)
	stapleFixes = strings.NewReplacer(
		"c/", " com ", "s/", " sem ",
	)
	// spelling and abbreviation variants seen in market listings
	stapleSynonyms = map[string]string{
		"parborizado": "parboilizado", "parbolizado": "parboilizado", "parboilisado": "parboilizado",
		"bco": "branco", "pto": "preto", "instataneo": "instantaneo", "inst": "instantaneo",
		"spaguetti": "espaguete", "espagueti": "espaguete", "semi": "semidesnatado",
		"desn": "desnatado", "integ": "integral", "int": "integral",
		"refin": "refinado",
	}
)

// foldName lowercases the name, drops accents and punctuation and spells out
// the common variants, so phrases can be looked up word for word.
func foldName(name string) string {
	s := stapleFixes.Replace(accents.Replace(strings.ToLower(name)))
	s = typeWordRe.ReplaceAllString(s, " tipo$1 ")
	words := strings.Fields(nonWordRe.ReplaceAllString(s, " "))
	for i, w := range words {
		if syn, ok := stapleSynonyms[w]; ok {
			words[i] = syn
		}
	}
	return " " + strings.Join(words, " ") + " "
}

// hasPhrase reports whether the whole words of phrase appear in folded.
func hasPhrase(folded, phrase string) bool {
	return strings.Contains(folded, " "+phrase+" ")
}

// classifyStaple tells which staple a product name describes; ok is false
// when it is not one (or can't be told apart from the others of its type).
func classifyStaple(name string) (staple, bool) {
	folded := foldName(name)
	// "Óleo de Soja", "Molho de Tomate": the "de" between type words is optional
	compact := strings.ReplaceAll(folded, " de ", " ")
	for _, st := range stapleTypes {
		if !startsWithAny(folded, compact, st.starts) {
			continue
		}
		for _, ex := range st.excludes {
			if hasPhrase(folded, ex) {
				return staple{}, false
			}
		}
		for word := range brandSensitiveWords {
			if hasPhrase(folded, word) {
				return staple{}, false
			}
		}
		parts := make([]string, 0, len(st.qualifiers))
		for _, q := range st.qualifiers {
			value := q.def
			for _, o := range q.options {
				if hasPhrase(folded, o.phrase) || hasPhrase(compact, o.phrase) {
					value = o.value
					break
				}
			}
			if value == "" {
				return staple{}, false
			}
			parts = append(parts, q.group+"="+value)
		}
		return staple{Type: st.name, Qualifiers: strings.Join(parts, ";")}, true
	}
	return staple{}, false
}

func startsWithAny(folded, compact string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(folded, " "+p+" ") || strings.HasPrefix(compact, " "+strings.ReplaceAll(p, " de ", " ")+" ") {
			return true
		}
	}
	return false
}

// sameSize: equivalents must come in (about) the same package, so the totals
// stay comparable: same kind of measure and pack count, and an amount within
// 10% ("1 kg" and "900 g", never "1 kg" and "5 kg").
func sameSize(a, b catalogRow) bool {
	if a.SizeKind != b.SizeKind || a.PackCount != b.PackCount || a.SoldByWeight != b.SoldByWeight {
		return false
	}
	if a.SizeAmount == 0 || b.SizeAmount == 0 {
		return a.SizeAmount == b.SizeAmount
	}
	ratio := a.SizeAmount / b.SizeAmount
	return ratio >= 0.9 && ratio <= 1/0.9
}

// isEquivalentStaple reports whether candidate can stand in for wanted: both
// the same staple, with the same qualifiers, in about the same package.
func isEquivalentStaple(wanted, candidate catalogRow) bool {
	w, ok := classifyStaple(wanted.Name)
	if !ok {
		return false
	}
	c, ok := classifyStaple(candidate.Name)
	if !ok || w.Type != c.Type || !sameSize(wanted, candidate) {
		return false
	}
	wRest, wClass := splitClass(w.Qualifiers)
	cRest, cClass := splitClass(c.Qualifiers)
	// the commercial class (tipo 1, 2, 3; 1 is the best grade) may be the
	// same or better, never worse: tipo 1 rice stands in for tipo 2
	return wRest == cRest && cClass <= wClass
}

// splitClass takes the "classe=N" qualifier out of q, returning the others
// and N ("" when the type has no class).
func splitClass(q string) (rest, class string) {
	parts := strings.Split(q, ";")
	kept := parts[:0]
	for _, p := range parts {
		if v, ok := strings.CutPrefix(p, "classe="); ok {
			class = v
			continue
		}
		kept = append(kept, p)
	}
	return strings.Join(kept, ";"), class
}
