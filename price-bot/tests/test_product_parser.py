from __future__ import annotations

from pathlib import Path

import pytest

from nextprice.product_parser import UnitPrice, fold, parse_product_name

CORPUS = Path(__file__).parent / "data" / "nomes_reais.txt"


# --- normalisation ---------------------------------------------------------------


def test_fold_removes_accents_case_and_apostrophes() -> None:
    assert fold("Açúcar Olho D`água  1KG") == "acucar olho dagua 1kg"
    assert fold("Hershey’s Ao Leite") == "hersheys ao leite"


# --- size ------------------------------------------------------------------------


@pytest.mark.parametrize(
    ("name", "kind", "count", "total", "label"),
    [
        ("Arroz Branco Camil Tipo 1 5kg", "mass", 1, 5000, "5 kg"),
        ("Achocolatado em Pó Nescau 350g", "mass", 1, 350, "350 g"),
        ("Maionese Heinz 390gr", "mass", 1, 390, "390 g"),
        ("Chocolate em Pó Sicao 50% Cacau 1,01kg", "mass", 1, 1010, "1,01 kg"),
        ("Cerveja Baden Baden Witbier 600ml", "volume", 1, 600, "600 ml"),
        ("Refrigerante Antarctica Guarana Zero 350 Ml Lata", "volume", 1, 350, "350 ml"),
        ("Leite Longa Vida Italac Com tampa Integral 1L", "volume", 1, 1000, "1 L"),
        ("Molho de Mostarda Cepêra 1,01L", "volume", 1, 1010, "1,01 L"),
        ("Cerveja Cusqueña Golden 330ml", "volume", 1, 330, "330 ml"),
        # NxW: the measure is per piece
        ("Maionese Hellmann's 2x500g", "mass", 2, 1000, "2 x 500 g"),
        ("Chocolate Nestlé Chokito 30x32g", "mass", 30, 960, "30 x 32 g"),
        ("Chocolate Tortuguita Arcor Baunilha 24x15,5g", "mass", 24, 372, "24 x 15,5 g"),
        ("Papel Higienico Mili 12x30m Folha Dupla Perfumado", "length", 12, 360, "12 x 30 m"),
        # measure + "N rolos": the measure is per roll
        ("Papel Higiênico Max Pure Folha dupla 30m 12 rolos", "length", 12, 360, "12 x 30 m"),
        # "N uni <weight>": the weight is the whole pack
        ("Ferrero Rocher 8 Uni 100g", "mass", 8, 100, "8 x 12,5 g"),
        ("Kinder Bueno Chocolate ao Leite 2 unis 43g", "mass", 2, 43, "2 x 21,5 g"),
        # width x length: only the length counts
        ("Folha Aluminio Wyda 30cm X 7,5m Rolinho", "length", 1, 7.5, "7,5 m"),
    ],
)
def test_size(name: str, kind: str, count: int, total: float, label: str) -> None:
    parsed = parse_product_name(name)
    assert parsed.unit_kind == kind
    assert parsed.pack_count == count
    assert parsed.total_amount == pytest.approx(total)
    assert parsed.size_label == label


@pytest.mark.parametrize(
    ("name", "count"),
    [
        ("Chocolate Baton Garoto Ao leite Display com 30 un", 30),
        ("Absorvente Sempre Livre Adapt Suave Com Abas Com 32", 32),
        ("Fralda Babysec Ultra Xg C/12", 12),
        ("Abacaxi Premium 1 Un", 1),
    ],
)
def test_count_only(name: str, count: int) -> None:
    parsed = parse_product_name(name)
    assert parsed.total_amount is None
    assert parsed.pack_count == count
    assert parsed.size_label == f"{count} un"


def test_measure_with_separate_count_is_per_piece() -> None:
    # "Capsula Chocolatto Tres 11g C/10": 10 capsules of 11g
    parsed = parse_product_name("Capsula Chocolatto Tres 11g C/10")
    assert (parsed.pack_count, parsed.unit_amount, parsed.total_amount) == (10, 11, 110)


def test_percentages_and_model_codes_are_not_sizes() -> None:
    cacau = parse_product_name("Chocolate em Pó Sicao 50% Cacau 200g")
    assert cacau.total_amount == 200
    assert "50%" in cacau.generic_name

    copo = parse_product_name("Copo Americano Nadir Ref. 2010 un")
    assert copo.pack_count == 1  # "2010 un" is a model code, not 2010 cups
    assert copo.total_amount is None


def test_sold_by_weight_and_approximate() -> None:
    queijo = parse_product_name("Queijo Mussarela Diversos Peça KG")
    assert queijo.sold_by_weight
    assert "kg" not in queijo.generic_name.split()

    banana = parse_product_name("Banana Prata Porção Aprox. 1kg")
    assert banana.approximate
    assert banana.total_amount == 1000


def test_name_without_any_size() -> None:
    parsed = parse_product_name("↪️ Atacado SUPER!")
    assert not parsed.has_size
    assert parsed.brand is None


# --- brand -----------------------------------------------------------------------


@pytest.mark.parametrize(
    ("name", "brand"),
    [
        ("Arroz Branco Camil Tipo 1 5kg", "Camil"),
        ("Cafe Melitta Tradicional Vacuo 250g", "Melitta"),
        ("Cerveja Baden Baden Witbier 600ml", "Baden Baden"),  # two-word brand
        ("Cerveja Stella Artois Pure Gold 269ml", "Stella Artois"),
        ("Maionese Hellmann’s 200g", "Hellmann's"),  # curly apostrophe
        ("Acucar Cristal Olho D`agua 1kg", "Olho D'água"),
        ("Detergente Pó Sch Tixan Ypê 800g Primaver", "Tixan Ypê"),  # longest match
        ("Chocolate Kit Kat Nestlé Ao Leite 41,5g", "Nestlé"),
        ("Cerveja Original 269ml", "Original"),
        ("Banana Prata Porção Aprox. 1kg", None),
    ],
)
def test_brand(name: str, brand: str | None) -> None:
    assert parse_product_name(name).brand == brand


def test_conditional_brand_only_applies_with_its_keyword() -> None:
    # "Original" is a beer brand, but also just a flavour of Bis
    assert parse_product_name("Chocolate Bis Original Ao Leite 100,8g").brand is None


# --- generic key (equivalence across brands) ----------------------------------------


def _key(name: str) -> str:
    return parse_product_name(name).generic_key


def test_same_generic_key_across_brands() -> None:
    assert _key("Arroz Branco Camil Tipo 1 5kg") == _key("Arroz Branco Sofia Tipo1 5kg")
    assert (
        _key("Feijão Carioca Kicaldo Tipo 1 1kg")
        == _key("Feijao Carioca Ideal Tipo-1 1kg")
        == _key("Feijao Carioca Tipo 1 Sofia 1kg")
    )
    assert _key("Cerveja Heineken 350ml Sleek Lt") == _key("Cerveja Heineken Sleek 350ml")
    assert _key("Refrigerante Antarctica Guarana Zero 350 Ml Lata") == _key(
        "Refrigerante Guaraná Antarctica Zero 200ml"
    )


def test_different_variants_keep_different_keys() -> None:
    # same brand, same size, different product: must NOT be considered equivalent
    assert _key("Chocolate Nestlé Alpino 80g") != _key("Chocolate Nestlé Galak 80g")
    assert _key("Ketchup Heinz Zero 530g") != _key("Ketchup Heinz Tradicional 567g")
    assert _key("Leite Condensado Moça Integral 395g") != _key(
        "Leite Condensado Moça Semidesnatado 395g"
    )
    assert _key("Arroz Branco Camil Tipo 1 5kg") != _key("Arroz Camil Integral 1kg")
    assert _key("Chocolate Ouro Branco 540g") != _key("Chocolate Sonho de Valsa 1kg")


def test_same_product_different_size_shares_key_but_not_amount() -> None:
    small = parse_product_name("Café Melitta Tradicional Vácuo 500g")
    big = parse_product_name("Cafe Melitta Tradicional Vacuo 250g")
    assert small.brand == big.brand == "Melitta"
    assert small.generic_key == big.generic_key
    assert small.total_amount != big.total_amount  # trigram similarity alone would merge them


def test_same_product_written_two_ways_has_same_amount() -> None:
    a = parse_product_name("Maionese Heinz 390g")
    b = parse_product_name("Maionese Heinz 390gr")
    assert (a.brand, a.generic_key, a.total_amount) == (b.brand, b.generic_key, b.total_amount)


# --- unit price ------------------------------------------------------------------


@pytest.mark.parametrize(
    ("name", "price", "expected"),
    [
        ("Café Melitta Tradicional Vácuo 500g", 28.90, UnitPrice(57.80, "kg")),
        ("Arroz Branco Camil Tipo 1 5kg", 22.69, UnitPrice(4.54, "kg")),
        ("Leite Piracanjuba Integral 1l", 6.89, UnitPrice(6.89, "L")),
        ("Cerveja Heineken 350ml Sleek Lt", 7.19, UnitPrice(20.54, "L")),
        ("Maionese Hellmann's 2x500g", 30.00, UnitPrice(30.00, "kg")),
        ("Papel Higienico Mili 12x30m Folha Dupla Perfumado", 17.90, UnitPrice(0.05, "m")),
        ("Fralda Babysec Ultra Xg C/12", 60.00, UnitPrice(5.00, "un")),
        ("Queijo Mussarela Diversos Peça KG", 39.90, UnitPrice(39.90, "kg")),
    ],
)
def test_unit_price(name: str, price: float, expected: UnitPrice) -> None:
    assert parse_product_name(name).unit_price(price) == expected


def test_unit_price_unknown_when_name_has_no_size() -> None:
    assert parse_product_name("Copo Americano Nadir Ref. 2010 un").unit_price(5.0) is None


# --- real corpus ---------------------------------------------------------------------


@pytest.fixture(scope="module")
def corpus() -> list[str]:
    return [n.strip() for n in CORPUS.read_text(encoding="utf-8").splitlines() if n.strip()]


def test_parses_every_real_name_without_crashing(corpus: list[str]) -> None:
    assert len(corpus) > 300
    for name in corpus:
        parsed = parse_product_name(name)
        assert parsed.raw_name == name
        if parsed.total_amount is not None:
            assert parsed.total_amount > 0
            assert parsed.unit_kind in {"mass", "volume", "length"}


def test_real_corpus_coverage_does_not_regress(corpus: list[str]) -> None:
    parsed = [parse_product_name(n) for n in corpus]
    sized = sum(1 for p in parsed if p.has_size or p.sold_by_weight)
    branded = sum(1 for p in parsed if p.brand)
    # measured at 97.9% / 90.4% when written; keep a margin for new corpus entries
    assert sized / len(parsed) >= 0.95
    assert branded / len(parsed) >= 0.85
