"""Preenche marca, tamanho, chave genérica etc. nos produtos que JÁ estão no
price-back, aplicando o parser sobre o nome de cada um.

Usa PATCH /api/products/{id} sem enviar preço, então a data de coleta de cada
preço fica como estava (um backfill não pode fazer preço antigo parecer novo).

Uso (na pasta price-bot):
    .venv/Scripts/python.exe -X utf8 scripts/backfill_parsed_fields.py [--dry-run]

Variável de ambiente: PRICE_BACK_URL (padrão http://localhost:8080).
Produtos que parecem banner de vitrine são apenas listados, nunca apagados.
"""

from __future__ import annotations

import os
import sys

import httpx

from nextprice.browser import looks_like_banner
from nextprice.product_parser import parse_product_name, price_back_fields

PRICE_BACK_URL = os.environ.get("PRICE_BACK_URL", "http://localhost:8080").rstrip("/")


def main() -> int:
    dry_run = "--dry-run" in sys.argv
    updated = skipped = failed = 0
    banners: list[dict] = []

    with httpx.Client(timeout=15) as client:
        products = client.get(f"{PRICE_BACK_URL}/api/products").raise_for_status().json()
        print(f"{len(products)} produtos em {PRICE_BACK_URL}")

        for product in products:
            if looks_like_banner(product["name"]):
                banners.append(product)
                continue

            fields = price_back_fields(parse_product_name(product["name"]))
            fields.setdefault("size_kind", "")  # name without size: clear a stale one

            if dry_run:
                skipped += 1
                continue
            resp = client.patch(f"{PRICE_BACK_URL}/api/products/{product['id']}", json=fields)
            if resp.status_code == 200:
                updated += 1
            else:
                failed += 1
                print(f"  falhou {product['name']!r}: {resp.status_code} {resp.text}")

    print(f"atualizados: {updated} | falhas: {failed} | só simulados: {skipped}")
    if banners:
        print(f"\n{len(banners)} produto(s) parecem banner de vitrine (NÃO apagados):")
        for b in banners:
            print(f"  {b['id']}  {b['name']!r}  R$ {b['default_price']}")
    return 1 if failed else 0


if __name__ == "__main__":
    raise SystemExit(main())
