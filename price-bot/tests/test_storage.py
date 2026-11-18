import json
from pathlib import Path

from nextprice.storage import dedupe_by_name, save_json


def test_dedupe_by_name_remove_duplicados() -> None:
    produtos = [
        {"nome": "Tomate", "preco": "R$ 5,00"},
        {"nome": "Tomate", "preco": "R$ 6,00"},
        {"nome": "Arroz", "preco": "R$ 20,00"},
    ]
    unicos = dedupe_by_name(produtos)
    assert len(unicos) == 2
    nomes = {p["nome"] for p in unicos}
    assert nomes == {"Tomate", "Arroz"}


def test_save_json_escreve_arquivo(tmp_path: Path) -> None:
    produtos = [{"nome": "Tomate", "preco": "R$ 5,00"}]
    caminho = save_json("LojaTeste", produtos, pasta=str(tmp_path))

    assert caminho is not None
    assert caminho.exists()
    conteudo = json.loads(caminho.read_text(encoding="utf-8"))
    assert conteudo == produtos


def test_save_json_lista_vazia_nao_escreve_arquivo(tmp_path: Path) -> None:
    caminho = save_json("LojaTeste", [], pasta=str(tmp_path))
    assert caminho is None
    assert list(tmp_path.iterdir()) == []
