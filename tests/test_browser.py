from __future__ import annotations

from nextprice.browser import looks_like_banner


def testlooks_like_banner_detects_emoji_prefixed_promo_text() -> None:
    assert looks_like_banner("🔷 Atacado SUPER!")
    assert looks_like_banner("✨ Da Nossa Família para Sua!")
    assert looks_like_banner("🏷️ Rótulos Selecionados para você")
    assert looks_like_banner("💝 Frutas de qualidades!")


def testlooks_like_banner_detects_emoji_anywhere_in_text() -> None:
    assert looks_like_banner("09.09 é AQUI ✓")


def testlooks_like_banner_detects_promo_phrases_without_emoji() -> None:
    assert looks_like_banner("Ofertas do dia")
    assert looks_like_banner("Oferta Exclusivas")
    assert looks_like_banner("Ofertas em Destaque")
    assert looks_like_banner("Mais Vendidos")
    assert looks_like_banner("Exclusivo Delivery")
    assert looks_like_banner("Peso Variável")


def testlooks_like_banner_matches_regardless_of_accents_or_case() -> None:
    assert looks_like_banner("PROMOCAO IMPERDIVEL")
    assert looks_like_banner("promoção imperdível")


def testlooks_like_banner_real_product_names_are_not_flagged() -> None:
    assert not looks_like_banner("Tomate Kg")
    assert not looks_like_banner("Whisky Black & White 1l")
    assert not looks_like_banner("Queijo Mussarela Diversos Peça KG")
    assert not looks_like_banner("Shampoo Turma Da Xuxa 210ml Camomila")


def test_looks_like_banner_detects_arrow_symbols_and_variation_selector() -> None:
    # regression: the real "↪️ Atacado SUPER!" banner got registered as a product
    assert looks_like_banner("↪️ Atacado SUPER!")
    assert looks_like_banner("➡ Confira")
    assert looks_like_banner("▶ Novidades")


def test_looks_like_banner_symbols_common_in_product_names_are_not_flagged() -> None:
    assert not looks_like_banner("Chocolate Hershey’s Ao Leite 82g")
    assert not looks_like_banner("Cerveja Corona Cero 330ml Ln 0% Alc")
    assert not looks_like_banner("Leite Condensado Milk + Semidesnatado 395g")
    assert not looks_like_banner("Maionese Hellmann's 2x500g")
    assert not looks_like_banner("Cerveja Corona Zero Álcool 330ml")
