from __future__ import annotations

import json
import logging
import os
from datetime import datetime
from pathlib import Path

logger = logging.getLogger(__name__)

PASTA_DADOS = "dados_supermercados"


def dedupe_by_name(produtos: list[dict]) -> list[dict]:
    """Remove duplicados mantendo a última ocorrência de cada nome de produto."""
    return list({item["nome"]: item for item in produtos}.values())


def save_json(nome_mercado: str, produtos: list[dict], pasta: str = PASTA_DADOS) -> Path | None:
    """Deduplica por nome e salva em um JSON datado. Retorna o caminho salvo, ou None se vazio."""
    unicos = dedupe_by_name(produtos)
    if not unicos:
        logger.warning("[%s] Nenhum produto encontrado.", nome_mercado)
        return None

    os.makedirs(pasta, exist_ok=True)
    arquivo = Path(pasta) / f"{nome_mercado.lower()}_{datetime.now().strftime('%Y-%m-%d')}.json"
    with open(arquivo, "w", encoding="utf-8") as f:
        json.dump(unicos, f, ensure_ascii=False, indent=4)

    logger.info("[%s] Sucesso! %d produtos salvos em '%s'.", nome_mercado, len(unicos), arquivo)
    return arquivo
