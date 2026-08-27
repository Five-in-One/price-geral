from __future__ import annotations


def parse_price_brl(preco: str) -> float:
    """Converte um preço no formato brasileiro (ex.: "R$ 7,79") para float (7.79)."""
    preco_limpo = preco.replace("R$", "").replace(" ", "").replace(".", "").replace(",", ".")
    return float(preco_limpo)
