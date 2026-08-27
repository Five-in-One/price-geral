from __future__ import annotations

import json

PROMPT_TEMPLATE = """
Analise os blocos de produtos abaixo extraídos da página de busca do supermercado
'{loja}' para o termo '{termo}'.
Extraia apenas os produtos válidos que correspondem ao item fresco ou in natura
pesquisado (ex: 'Tomate'), ignorando molhos ou itens que não sejam o fruto principal.
Retorne a resposta EXATAMENTE no formato de uma lista JSON válida de objetos, onde
cada objeto tem as chaves "nome" e "preco". Exemplo:
[
  {{"nome": "Tomate Longa Vida 1kg", "preco": "R$ 7,79"}}
]
Se não encontrar nenhum produto, retorne [].

Blocos de Produtos:
{blocos}
"""


def build_prompt(loja: str, termo: str, card_texts: list[str]) -> str:
    """Monta o prompt enviado à IA a partir dos blocos de texto coletados na página."""
    blocos = "\n---\n".join(card_texts)
    return PROMPT_TEMPLATE.format(loja=loja, termo=termo, blocos=blocos)


def parse_ai_response(raw_text: str) -> list[dict]:
    """Valida e converte a resposta JSON da IA em uma lista de produtos.

    Levanta ValueError com uma mensagem clara se a resposta não for um JSON
    válido ou não for uma lista, em vez de deixar o erro estourar como
    json.JSONDecodeError/TypeError genérico para quem chamou.
    """
    try:
        dados = json.loads(raw_text)
    except json.JSONDecodeError as exc:
        raise ValueError(f"Resposta da IA não é um JSON válido: {exc}") from exc

    if not isinstance(dados, list):
        raise ValueError(f"Resposta da IA deveria ser uma lista, veio {type(dados).__name__}")

    return dados
