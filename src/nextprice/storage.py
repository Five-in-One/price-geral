from __future__ import annotations

import json
import logging
import os
from datetime import datetime
from pathlib import Path

import psycopg

from nextprice.pricing import parse_price_brl

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


def save_to_postgres(
    conn: psycopg.Connection,
    store_key: str,
    store_name: str,
    produtos: list[dict],
    source: str,
    search_term: str | None = None,
) -> int:
    """Inserts raw scraped products into scraped_products.

    Each row keeps the original price text alongside a best-effort parsed
    numeric value (NULL when parsing fails, e.g. "sob consulta") — turning
    this raw data into price-back's curated markets/products schema is a
    separate step, not something the scraper decides on its own.

    Returns how many rows were inserted.
    """
    rows: list[tuple] = []
    for item in produtos:
        nome = item.get("nome")
        preco_texto = item.get("preco")
        if not nome or not preco_texto:
            continue

        try:
            preco = parse_price_brl(preco_texto)
        except (ValueError, AttributeError):
            preco = None

        rows.append((store_key, store_name, nome, preco_texto, preco, source, search_term))

    if not rows:
        return 0

    with conn.cursor() as cur:
        cur.executemany(
            """
            INSERT INTO scraped_products
                (store_key, store_name, product_name, price_text, price, source, search_term)
            VALUES (%s, %s, %s, %s, %s, %s, %s)
            """,
            rows,
        )
    conn.commit()

    logger.info("[%s] %d produtos salvos no Postgres (source=%s).", store_name, len(rows), source)
    return len(rows)
