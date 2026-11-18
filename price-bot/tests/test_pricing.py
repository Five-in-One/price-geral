import pytest

from nextprice.pricing import parse_price_brl


@pytest.mark.parametrize(
    "preco, esperado",
    [
        ("R$ 7,79", 7.79),
        ("R$7,79", 7.79),
        ("R$ 1.234,56", 1234.56),
        ("R$ 0,99", 0.99),
        ("R$ 10,00", 10.0),
    ],
)
def test_parse_price_brl(preco: str, esperado: float) -> None:
    assert parse_price_brl(preco) == pytest.approx(esperado)


def test_parse_price_brl_invalido_levanta_erro() -> None:
    with pytest.raises(ValueError):
        parse_price_brl("preço indisponível")
