from __future__ import annotations

from nextprice.browser import _looks_like_banner


def test_looks_like_banner_detects_emoji_prefixed_promo_text() -> None:
    assert _looks_like_banner("🔷 Atacado SUPER!")
    assert _looks_like_banner("✨ Da Nossa Família para Sua!")
    assert _looks_like_banner("🏷️ Rótulos Selecionados para você")
    assert _looks_like_banner("💝 Frutas de qualidades!")


def test_looks_like_banner_detects_emoji_anywhere_in_text() -> None:
    assert _looks_like_banner("09.09 é AQUI ✓")


def test_looks_like_banner_detects_promo_phrases_without_emoji() -> None:
    assert _looks_like_banner("Ofertas do dia")
    assert _looks_like_banner("Oferta Exclusivas")
    assert _looks_like_banner("Ofertas em Destaque")
    assert _looks_like_banner("Mais Vendidos")
    assert _looks_like_banner("Exclusivo Delivery")
    assert _looks_like_banner("Peso Variável")


def test_looks_like_banner_matches_regardless_of_accents_or_case() -> None:
    assert _looks_like_banner("PROMOCAO IMPERDIVEL")
    assert _looks_like_banner("promoção imperdível")


def test_looks_like_banner_real_product_names_are_not_flagged() -> None:
    assert not _looks_like_banner("Tomate Kg")
    assert not _looks_like_banner("Whisky Black & White 1l")
    assert not _looks_like_banner("Queijo Mussarela Diversos Peça KG")
    assert not _looks_like_banner("Shampoo Turma Da Xuxa 210ml Camomila")
