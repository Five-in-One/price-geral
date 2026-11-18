from __future__ import annotations

import logging

from google import genai
from google.genai import types

from nextprice.ai.prompts import (
    build_classification_prompt,
    build_generic_extraction_prompt,
    build_prompt,
    parse_ai_response,
)
from nextprice.config import Settings

logger = logging.getLogger(__name__)

_MODEL = "gemini-3.6-flash"


class GeminiClient:
    def __init__(self, settings: Settings | None = None) -> None:
        settings = settings or Settings.load()
        self._client = genai.Client(api_key=settings.gemini_api_key)

    def extract_products(self, loja: str, termo: str, card_texts: list[str]) -> list[dict]:
        """Envia os blocos de texto coletados na página para a IA extrair produtos estruturados."""
        if not card_texts:
            return []

        prompt = build_prompt(loja, termo, card_texts)
        try:
            response = self._client.models.generate_content(
                model=_MODEL,
                contents=prompt,
                config=types.GenerateContentConfig(
                    response_mime_type="application/json",
                    temperature=0.1,
                ),
            )
            if response.text is None:
                raise ValueError("Resposta da IA veio vazia (sem texto).")
            produtos = parse_ai_response(response.text)
        except Exception:
            logger.warning("[%s] Falha ao processar resultados com a IA.", loja, exc_info=True)
            return []

        for item in produtos:
            item["loja"] = loja
        return produtos

    def classify_categories(self, loja: str, nomes_produtos: list[str]) -> dict[str, dict]:
        """Como extract_products_generic, mas para produtos cujo nome/preço
        já vieram de seletores CSS (loja conhecida): pede à IA só
        categoria/unidade, uma chamada mais leve. Retorna um dict
        nome -> {categoria, unidade}; dict vazio se a IA falhar, para o
        chamador aplicar um default em vez de descartar o produto."""
        if not nomes_produtos:
            return {}

        prompt = build_classification_prompt(loja, nomes_produtos)
        try:
            response = self._client.models.generate_content(
                model=_MODEL,
                contents=prompt,
                config=types.GenerateContentConfig(
                    response_mime_type="application/json",
                    temperature=0.1,
                ),
            )
            if response.text is None:
                raise ValueError("Resposta da IA veio vazia (sem texto).")
            classificacoes = parse_ai_response(response.text)
        except Exception:
            logger.warning("[%s] Falha ao classificar categorias com a IA.", loja, exc_info=True)
            return {}

        return {
            item["nome"]: item
            for item in classificacoes
            if isinstance(item, dict) and "nome" in item
        }

    def extract_products_generic(self, loja: str, card_texts: list[str]) -> list[dict]:
        """Como extract_products, mas sem termo de busca: usada pela coleta
        sob demanda do painel administrativo, onde a página é desconhecida e
        a IA também estima categoria/unidade de cada produto."""
        if not card_texts:
            return []

        prompt = build_generic_extraction_prompt(loja, card_texts)
        try:
            response = self._client.models.generate_content(
                model=_MODEL,
                contents=prompt,
                config=types.GenerateContentConfig(
                    response_mime_type="application/json",
                    temperature=0.1,
                ),
            )
            if response.text is None:
                raise ValueError("Resposta da IA veio vazia (sem texto).")
            produtos = parse_ai_response(response.text)
        except Exception:
            logger.warning("[%s] Falha ao processar coleta genérica com a IA.", loja, exc_info=True)
            return []

        for item in produtos:
            item["loja"] = loja
        return produtos
