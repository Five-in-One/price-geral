from __future__ import annotations

import logging

from google import genai
from google.genai import types

from nextprice.ai.prompts import build_prompt, parse_ai_response
from nextprice.config import Settings

logger = logging.getLogger(__name__)

_MODEL = "gemini-2.5-flash"


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
