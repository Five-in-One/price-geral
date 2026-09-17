from __future__ import annotations

from nextprice.categorize import classify_by_keywords, guess_unit


def test_classify_by_keywords_hortifruti() -> None:
    assert classify_by_keywords("Banana Prata Kg") == "HORTIFRUTI"
    assert classify_by_keywords("Tomate Kg") == "HORTIFRUTI"
    assert classify_by_keywords("Limão Taiti Bandeja 500g") == "HORTIFRUTI"


def test_classify_by_keywords_graos() -> None:
    assert classify_by_keywords("Arroz Branco Lf T1 Sofia 1kg") == "GRÃOS"
    assert classify_by_keywords("Feijão Preto T-1 Sofia 1kg") == "GRÃOS"
    assert classify_by_keywords("Cafe 3 Coracoes Tradicional Vacuo 250g") == "GRÃOS"


def test_classify_by_keywords_laticinios() -> None:
    assert classify_by_keywords("Leite Piracanjuba Integral 1l") == "LATICÍNIOS"
    assert classify_by_keywords("Queijo Mussarela Diversos Peça KG") == "LATICÍNIOS"
    assert classify_by_keywords("Bebida Láctea Nescau 180ml") == "LATICÍNIOS"


def test_classify_by_keywords_carnes() -> None:
    assert classify_by_keywords("Carne Bovina Bife Patinho 1kg") == "CARNES"
    assert classify_by_keywords("Coxa Sobre Coxa Aurora Kg Cong") == "CARNES"


def test_classify_by_keywords_no_match_falls_back_to_outros() -> None:
    assert classify_by_keywords("Whisky Black & White 1l") == "OUTROS"
    assert classify_by_keywords("Shampoo Turma Da Xuxa 210ml Camomila") == "OUTROS"


def test_classify_by_keywords_does_not_match_substring_inside_other_word() -> None:
    # "pera" (HORTIFRUTI) must not match inside "temPERAda".
    assert classify_by_keywords("Coxa S/coxa Frango Congelada Temperada Kg") == "CARNES"


def test_guess_unit_from_embedded_token() -> None:
    assert guess_unit("Banana Prata Kg") == "kg"
    assert guess_unit("Agua Mineral Crystal Com Gas 500ml") == "ml"
    assert guess_unit("Whisky Black & White 1l") == "l"
    assert guess_unit("Coco Verde Un") == "un"


def test_guess_unit_takes_last_token_when_multiple_present() -> None:
    assert guess_unit("Kit Sabonete 90g C/12 Un") == "un"


def test_guess_unit_no_token_defaults_to_un() -> None:
    assert guess_unit("Produto Sem Unidade No Nome") == "un"
