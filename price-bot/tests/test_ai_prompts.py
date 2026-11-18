import json

import pytest

from nextprice.ai.prompts import (
    CATEGORIES,
    build_classification_prompt,
    build_generic_extraction_prompt,
    build_prompt,
    parse_ai_response,
)


def test_build_prompt_inclui_loja_termo_e_blocos() -> None:
    prompt = build_prompt(
        "LojaTeste", "tomate", ["Tomate 1kg - R$ 7,79", "Molho de tomate - R$ 3,50"]
    )
    assert "LojaTeste" in prompt
    assert "tomate" in prompt
    assert "Tomate 1kg - R$ 7,79" in prompt
    assert "Molho de tomate - R$ 3,50" in prompt


def test_parse_ai_response_json_valido() -> None:
    raw = json.dumps([{"nome": "Tomate 1kg", "preco": "R$ 7,79"}])
    dados = parse_ai_response(raw)
    assert dados == [{"nome": "Tomate 1kg", "preco": "R$ 7,79"}]


def test_parse_ai_response_json_invalido_levanta_value_error() -> None:
    with pytest.raises(ValueError, match="JSON válido"):
        parse_ai_response("isso não é json")


def test_parse_ai_response_nao_lista_levanta_value_error() -> None:
    with pytest.raises(ValueError, match="lista"):
        parse_ai_response(json.dumps({"nome": "Tomate"}))


def test_build_generic_extraction_prompt_inclui_loja_categorias_e_blocos() -> None:
    prompt = build_generic_extraction_prompt("MercadoDesconhecido", ["Arroz 5kg - R$ 23,90"])
    assert "MercadoDesconhecido" in prompt
    assert "Arroz 5kg - R$ 23,90" in prompt
    for categoria in CATEGORIES:
        assert categoria in prompt


def test_build_generic_extraction_prompt_com_lista_vazia_nao_quebra() -> None:
    prompt = build_generic_extraction_prompt("Loja", [])
    assert "Loja" in prompt
    assert "Blocos de Produtos" in prompt


def test_build_classification_prompt_inclui_loja_categorias_e_nomes() -> None:
    prompt = build_classification_prompt("CarvalhoSuperShop", ["Tomate 1kg", "Arroz 5kg"])
    assert "CarvalhoSuperShop" in prompt
    assert "Tomate 1kg" in prompt
    assert "Arroz 5kg" in prompt
    for categoria in CATEGORIES:
        assert categoria in prompt


def test_build_classification_prompt_nao_pede_preco() -> None:
    prompt = build_classification_prompt("Loja", ["Tomate 1kg"])
    assert '"preco"' not in prompt
