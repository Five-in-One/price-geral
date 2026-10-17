package repository

import "testing"

// Real listing names (FerreiraEmCasa, CarvalhoSuperShop, Atacadão) and what
// they are.
func TestClassifyStaple(t *testing.T) {
	cases := []struct {
		name     string
		wantType string
		wantQual string
	}{
		{"Arroz Branco Dona Elza 5kg Tp2", "arroz", "tipo=branco;classe=2"},
		{"Arroz Tio Joao Tipo 1 5kg", "arroz", "tipo=branco;classe=1"}, // "branco" is implicit
		{"Arroz B Lf T1 Bellsabor 5kg", "arroz", "tipo=branco;classe=1"},
		{"Arroz Parborizado Meu Biju 5kg Tp1", "arroz", "tipo=parboilizado;classe=1"},
		{"Arroz Camil Integral 1kg", "arroz", "tipo=integral;classe=1"},
		{"Feijao Sempre Verde Popy 1kg Tp2", "feijao", "variedade=sempre verde;classe=2"},
		{"Feijao Sempre Verde Ideal Tipo-2 1kg", "feijao", "variedade=sempre verde;classe=2"},
		{"Feijao Branco Fradinho T-2 Ideal 1kg", "feijao", "variedade=fradinho;classe=2"},
		{"Feijão Carioca Kicaldo Tipo 1 1kg", "feijao", "variedade=carioca;classe=1"},
		{"Feijao De Corda Kicaldo 1kg", "feijao", "variedade=corda;classe=1"},
		{"Açúcar Caravelas Refinado 1kg", "acucar", "tipo=refinado"},
		{"Acucar Cristal Olho D`agua 1kg", "acucar", "tipo=cristal"},
		{"Sal Cisne Refinado Extra 1kg", "sal", "tipo=refinado"},
		{"Óleo de Soja Soya 900ml", "oleo", "tipo=soja"},
		{"Oleo Soja Sinha 900ml", "oleo", "tipo=soja"},
		{"Leite Longa Vida Italac Com tampa Integral 1L", "leite", "gordura=integral;lactose=normal"},
		{"Leite Piracanjuba Integral 1l", "leite", "gordura=integral;lactose=normal"},
		{"Leite Longa Vida Uht Piracanjuba 1l Desnatado Zero Lactose", "leite", "gordura=desnatado;lactose=zero"},
		{"Leite Condensado Moça Integral 395g", "leite condensado", "gordura=integral;lactose=normal"},
		{"Leite Condensado Itambé Semidesnatado 395g", "leite condensado", "gordura=semidesnatado;lactose=normal"},
		{"Creme de Leite Nestlé 200g", "creme de leite", "gordura=normal;lactose=normal"},
		{"Farinha De Trigo Dona Benta Com Fermento 1kg", "farinha de trigo", "fermento=com;tipo=comum"},
		{"Farinha Trigo Rosa Branca 1kg", "farinha de trigo", "fermento=sem;tipo=comum"},
		{"Farinha Mandioca Do Para Sofia 1kg Amarela", "farinha de mandioca", "cor=amarela;textura=comum"},
		{"Flocão de Milho Maratá 500g", "flocao de milho", ""},
		{"Farinha De Milho Flocada Para Cuscuz Coringa 500g", "flocao de milho", ""},
		{"Macarrao Fortaleza Parafuso Com Ovos 500g", "macarrao", "formato=parafuso;massa=ovos"},
		{"Macarrão com Ovos Dona Benta Espaguete 500g", "macarrao", "formato=espaguete;massa=ovos"},
		{"Molho De Tomate Tambau 300g Manjericao", "molho de tomate", "sabor=manjericao"},
		{"Molho de Tomate Tomadoro 300g", "molho de tomate", "sabor=tradicional"},
		{"Vinagre de Álcool Vitalia 750ml", "vinagre", "tipo=alcool"},
		{"Papel Higiênico Max Pure Folha dupla 30m 12 rolos", "papel higienico", "folha=dupla"},
	}
	for _, c := range cases {
		got, ok := classifyStaple(c.name)
		if !ok {
			t.Errorf("classifyStaple(%q): not a staple, want %s", c.name, c.wantType)
			continue
		}
		if got.Type != c.wantType || got.Qualifiers != c.wantQual {
			t.Errorf("classifyStaple(%q) = %s [%s], want %s [%s]", c.name, got.Type, got.Qualifiers, c.wantType, c.wantQual)
		}
	}
}

// Products whose name starts like a staple but are something else, or whose
// brand is the point: never swapped.
func TestClassifyStaple_NotStaples(t *testing.T) {
	for _, name := range []string{
		"Leite de Coco Menina 200ml",
		"Leite Fermentado Batavo Bob Esponja 80g",
		"Leite em Pó Ninho Integral 380g",
		"Doce De Leite Massa Lili Vd 680g",
		"Chocolate Ao Leite Lacta 80g",
		"Sal para churrasco Lebre 1kg",
		"Sal Churrasco Mariza Temperado 1kg",
		"Macarrão Instantâneo Nissin Lámen Galinha caipira 85g",
		"Tempero Sazón Feijão 60g",
		"Sardinha Gomes Da Costa Molho De Tomate 125g",
		"Óleo Amendoas Paixão Tentadora 200ml",
		"Refrigerante Coca-Cola Sem açúcar 200ml",
		"Cafe Melitta Tradicional Vacuo 250g", // coffee stays brand-loyal
		"Cerveja Heineken 350ml",
		"Vinho Tinto Quinta Do Morgado 750ml",
		"Feijao Kicaldo 1kg", // which bean? can't tell
		"Detergente Pó Sch Tixan Ypê 800g Primaver",
		"Vinagre Minhoto Alcool Aromatizado Alho 500ml",
		"Agua H2o 500ml Limao",                       // flavoured soft drink
		"Sal Parrila Br Spices Com Chimichuri 1010g", // seasoned salt
	} {
		if got, ok := classifyStaple(name); ok {
			t.Errorf("classifyStaple(%q) = %s [%s], want not a staple", name, got.Type, got.Qualifiers)
		}
	}
}

func TestIsEquivalentStaple(t *testing.T) {
	row := func(name string, kind string, amount float64) catalogRow {
		return catalogRow{Name: name, SizeKind: kind, SizeAmount: amount, PackCount: 1}
	}
	cases := []struct {
		a, b catalogRow
		want bool
	}{
		// other brand, same thing, same size
		{row("Feijao Sempre Verde Popy 1kg Tp2", "mass", 1000), row("Feijao Sempre Verde Ideal Tipo-2 1kg", "mass", 1000), true},
		{row("Oleo Soja Sinha 900ml", "volume", 900), row("Óleo de Soja Soya 900ml", "volume", 900), true},
		{row("Arroz Tio Joao Tipo 1 5kg", "mass", 5000), row("Arroz Branco Camil Tipo 1 5kg", "mass", 5000), true},
		{row("Leite Piracanjuba Integral 1l", "volume", 1000), row("Leite Longa Vida Italac Com tampa Integral 1L", "volume", 1000), true},
		// about the same size: within 10%
		{row("Flocao De Milho Ideal 400g Livre Transg", "mass", 400), row("Flocão de Milho Maratá 500g", "mass", 500), false},
		{row("Óleo de Soja Soya 900ml", "volume", 900), row("Oleo de Soja Liza 1l", "volume", 1000), true},
		// a qualifier differs
		{row("Arroz Branco Dona Elza 5kg Tp2", "mass", 5000), row("Arroz Parboilizado Alianca Tipo1 5kg", "mass", 5000), false},
		{row("Feijão Carioca Kicaldo Tipo 1 1kg", "mass", 1000), row("Feijão Preto T-1 Sofia 1kg", "mass", 1000), false},
		{row("Leite Piracanjuba Integral 1l", "volume", 1000), row("Leite Piracanjuba Desnatado 1l", "volume", 1000), false},
		{row("Molho De Tomate Tambau 300g Manjericao", "mass", 300), row("Molho de Tomate Tomadoro 300g", "mass", 300), false},
		// a better class stands in for a worse one, never the other way
		{row("Arroz Branco Dona Elza 5kg Tp2", "mass", 5000), row("Arroz Branco Catarinao Tipo1 5kg", "mass", 5000), true},
		{row("Arroz Branco Catarinao Tipo1 5kg", "mass", 5000), row("Arroz Branco Dona Elza 5kg Tp2", "mass", 5000), false},
		{row("Feijao Preto Rio Maria 1kg Tp2", "mass", 1000), row("Feijão Preto T-1 Sofia 1kg", "mass", 1000), true},
		// another size
		{row("Arroz Branco Camil Tipo 1 5kg", "mass", 5000), row("Arroz Branco Sofia Tipo1 1kg", "mass", 1000), false},
	}
	for _, c := range cases {
		if got := isEquivalentStaple(c.a, c.b); got != c.want {
			t.Errorf("isEquivalentStaple(%q, %q) = %v, want %v", c.a.Name, c.b.Name, got, c.want)
		}
	}
}
