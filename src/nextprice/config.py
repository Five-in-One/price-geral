from __future__ import annotations

import os
from dataclasses import dataclass

from dotenv import find_dotenv, load_dotenv


@dataclass(frozen=True)
class Settings:
    gemini_api_key: str

    @classmethod
    def load(cls) -> Settings:
        # usecwd=True: procura o .env a partir do diretório onde o comando foi
        # executado, não de onde o pacote está instalado.
        load_dotenv(find_dotenv(usecwd=True))
        gemini_api_key = os.environ.get("GEMINI_API_KEY", "").strip()
        if not gemini_api_key:
            raise RuntimeError(
                "GEMINI_API_KEY não configurada. Defina-a no arquivo .env "
                "(veja .env.example) antes de rodar o querytracker."
            )
        return cls(gemini_api_key=gemini_api_key)
