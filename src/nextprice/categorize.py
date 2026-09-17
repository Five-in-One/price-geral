from __future__ import annotations

import re

from nextprice.text import normalize

# Categoriza por palavras-chave no nome — não depende da IA, que tem se
# mostrado pouco confiável (503 recorrente do Gemini às vezes bloqueia a
# classificação inteira). Cobre o vocabulário observado nos mercados já
# coletados; o que não bate cai em OUTROS, mesmo comportamento de antes.
# Comparado sem acento (ver nextprice.browser._normalize).
_CATEGORY_KEYWORDS: dict[str, tuple[str, ...]] = {
    "HORTIFRUTI": (
        "banana",
        "tomate",
        "cebola",
        "batata",
        "laranja",
        "limao",
        "manga",
        "mamao",
        "melancia",
        "melao",
        "kiwi",
        "pera",
        "maracuja",
        "alface",
        "goiaba",
        "uva",
        "abacaxi",
        "morango",
        "cenoura",
        "chuchu",
        "abobrinha",
        "couve",
        "repolho",
    ),
    "GRÃOS": (
        "arroz",
        "feijao",
        "farinha",
        "macarrao",
        "cafe",
        "flocao",
        "milho",
        "goma",
        "acucar",
        "oleo",
        "amendoim",
        "biscoito",
        "pipoca",
        "aveia",
        "granola",
    ),
    "LATICÍNIOS": (
        "leite",
        "queijo",
        "iogurte",
        "manteiga",
        "requeijao",
        "nescau",
        "achocolatado",
        "lactea",
    ),
    "CARNES": (
        "carne",
        "frango",
        "bovina",
        "bovino",
        "coxa",
        "peixe",
        "linguica",
        "salsicha",
        "bacon",
        "peito",
        "file",
        "picanha",
        "costela",
        "lombo",
    ),
}


# \b (fronteira de palavra) evita que uma palavra-chave curta bata dentro
# de outra palavra por coincidência — ex.: "pera" (HORTIFRUTI) não deve
# casar com "temPERAda" em "Frango Congelada Temperada".
_CATEGORY_PATTERNS: dict[str, re.Pattern[str]] = {
    categoria: re.compile(r"\b(?:" + "|".join(palavras) + r")\b")
    for categoria, palavras in _CATEGORY_KEYWORDS.items()
}


def classify_by_keywords(nome: str) -> str:
    """Classifica um produto em uma das CATEGORIES por palavra-chave no
    nome; sem correspondência, cai em "OUTROS"."""
    normalizado = normalize(nome)
    for categoria, padrao in _CATEGORY_PATTERNS.items():
        if padrao.search(normalizado):
            return categoria
    return "OUTROS"


# Unidade embutida no próprio nome do produto ("Banana Prata Kg", "Agua
# Mineral 500ml", "Whisky Black & White 1l") — mais confiável que pedir pra
# IA estimar, já que o dado já está ali. Pega o último token de unidade no
# nome (o mais próximo do fim costuma ser o que realmente vale).
_UNIT_TOKEN_PATTERN = re.compile(r"\b\d*[.,]?\d*\s*(kg|g|ml|l|un|pct)\b", re.IGNORECASE)


def guess_unit(nome: str) -> str:
    """Extrai a unidade de venda do nome do produto; "un" se não achar."""
    matches = list(_UNIT_TOKEN_PATTERN.finditer(nome))
    if not matches:
        return "un"
    return matches[-1].group(1).lower()
