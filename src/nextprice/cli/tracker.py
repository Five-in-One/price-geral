from __future__ import annotations

import logging

from playwright.sync_api import sync_playwright

from nextprice.ai.client import GeminiClient
from nextprice.browser import search_and_collect_cards
from nextprice.geo import get_user_location
from nextprice.pricing import parse_price_brl
from nextprice.stores.base import StoreConfig
from nextprice.stores.registry import STORES

logging.basicConfig(level=logging.INFO, format="%(message)s")
logger = logging.getLogger(__name__)


def buscar_na_loja(store: StoreConfig, termo: str, ai_client: GeminiClient) -> list[dict]:
    print(f"\n🛒 Acessando {store.name} e buscando por '{termo}'...")
    cards_textos: list[str] = []
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=False)
        page = browser.new_page()
        try:
            cards_textos = search_and_collect_cards(page, store, termo)
            print(
                f"[{store.name}] Encontrados {len(cards_textos)} blocos isolados de produtos. "
                "Enviando para a IA..."
            )
        except Exception as e:
            print(f"[{store.name}] Erro durante a automação: {e}")
        finally:
            browser.close()

    return ai_client.extract_products(store.name, termo, cards_textos)


def main() -> None:
    termo = input("O que você deseja pesquisar? (Ex: tomate, arroz): ").strip()
    if not termo:
        return

    localizacao = get_user_location()
    print(f"🌍 Localização detectada: {localizacao}")

    ai_client = GeminiClient()

    todos_resultados: list[dict] = []
    for store in STORES:
        produtos_loja = buscar_na_loja(store, termo, ai_client)
        if produtos_loja:
            todos_resultados.extend(produtos_loja)

    print("\n" + "=" * 50)
    print(f" COMPARATIVO INTELIGENTE PARA: '{termo.upper()}' ({localizacao})")
    print("=" * 50)

    if not todos_resultados:
        print("Nenhum produto retornado pelas lojas.")
        return

    validados = []
    for r in todos_resultados:
        try:
            r["valor_numerico"] = parse_price_brl(r["preco"])
            validados.append(r)
        except Exception:
            continue

    validados.sort(key=lambda x: x["valor_numerico"])

    for i, r in enumerate(validados):
        tag_vencedor = " 👑 [MAIS BARATO]" if i == 0 else ""
        print(f"🏪 Loja:    {r.get('loja')}{tag_vencedor}")
        print(f"📦 Produto: {r.get('nome')}")
        print(f"💰 Preço:   {r.get('preco')}\n")


if __name__ == "__main__":
    main()
