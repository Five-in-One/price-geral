from __future__ import annotations

import unicodedata


def normalize(texto: str) -> str:
    """Remove acentos e caixa para comparação de palavras-chave robusta a
    variações de grafia ("Promoção" / "Promocao" / "PROMOÇÃO")."""
    decomposto = unicodedata.normalize("NFKD", texto)
    return "".join(c for c in decomposto if not unicodedata.combining(c)).lower()
