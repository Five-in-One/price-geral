"""HTTP API used by the Admin panel's "coleta sob demanda": given a market
URL, scrape it, let the AI classify the products found, save the raw
result, and push the classified products into price-back.

Run with: `uvicorn nextprice.collect_api:app --port 8085` (or
`python -m nextprice.collect_api`).
"""

from __future__ import annotations

import logging
import os
from datetime import datetime, timezone

import httpx
from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from playwright.sync_api import sync_playwright
from pydantic import BaseModel, Field

from nextprice import db, storage
from nextprice.ai.client import GeminiClient
from nextprice.browser import (
    collect_listing_text_blocks,
    looks_like_banner,
    scrape_listing,
    scroll_to_load,
)
from nextprice.categorize import classify_by_keywords, guess_unit
from nextprice.pricing import parse_price_brl
from nextprice.product_parser import parse_product_name, price_back_fields
from nextprice.stores.base import StoreConfig
from nextprice.stores.registry import find_store_by_url

logging.basicConfig(level=logging.INFO, format="%(message)s")
logger = logging.getLogger(__name__)

PRICE_BACK_URL = os.environ.get("PRICE_BACK_URL", "http://localhost:8080").rstrip("/")

app = FastAPI(title="NextPrice Bot - Coleta sob Demanda")

# The Admin panel (a browser page) calls this API directly. No
# login/cookies here yet, so a permissive origin list is fine for this dev
# API — same reasoning as price-back's CORS setup.
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["*"],
    allow_headers=["*"],
)


class CollectRequest(BaseModel):
    url: str
    market_name: str = Field(min_length=1)
    # Address fields are optional and all default to "" — the Admin panel
    # sends whatever it has (CEP-autofilled or hand-typed), and price-back
    # merges non-empty values into the market's stored address without
    # blanking out what a previous submission already saved. This is the
    # one shape every future market registration goes through, regardless
    # of which store-specific (or generic AI) scraper ends up handling its
    # products.
    cep: str = ""
    street: str = ""
    number: str = ""
    neighborhood: str = ""
    city: str = ""
    state: str = ""
    complement: str = ""


class CollectedProduct(BaseModel):
    name: str
    price: float
    category: str
    unit: str


class CollectResponse(BaseModel):
    market_name: str
    blocks_found: int
    products_extracted: int
    products_inserted: int
    products: list[CollectedProduct]


def scrape_blocks(url: str) -> list[str]:
    """Opens url headlessly and collects text blocks that look like
    product cards. Split out for testability — tests monkeypatch this
    instead of driving a real browser."""
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        page = browser.new_page()
        try:
            return collect_listing_text_blocks(page, url)
        finally:
            browser.close()


def scrape_known_store(url: str, store: StoreConfig) -> list[dict]:
    """Structured scrape for a store with registered CSS selectors: gets
    exact nome/preco pairs straight from the page, without relying on the
    AI to interpret raw text (unlike scrape_blocks, used for unregistered
    markets). Split out for testability, like scrape_blocks."""
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        page = browser.new_page()
        try:
            page.goto(url, timeout=60000)
            page.wait_for_timeout(5000)
            scroll_to_load(page)
            return scrape_listing(page, store)
        finally:
            browser.close()


def classify_products(market_name: str, produtos_brutos: list[dict]) -> list[dict]:
    """Enriches CSS-scraped nome/preco pairs with categoria/unidade.
    Prefers the AI's classification when it succeeds, but falls back to
    classify_by_keywords/guess_unit (nextprice.categorize) instead of a
    blind "OUTROS"/"un" default — the AI (Gemini) has been failing often
    enough that a local, dependency-free classifier is the more reliable
    default, not just a last resort."""
    if not produtos_brutos:
        return []

    ai_client = GeminiClient()
    classificacoes = ai_client.classify_categories(
        market_name, [p["nome"] for p in produtos_brutos]
    )

    produtos = []
    for item in produtos_brutos:
        extra = classificacoes.get(item["nome"], {})
        produtos.append(
            {
                "nome": item["nome"],
                "preco": item["preco"],
                "categoria": extra.get("categoria") or classify_by_keywords(item["nome"]),
                "unidade": extra.get("unidade") or guess_unit(item["nome"]),
                "loja": market_name,
            }
        )
    return produtos


def save_raw_scrape(market_name: str, url: str, produtos: list[dict]) -> None:
    """Best-effort: keeps the raw AI output in price-bot's own staging
    table regardless of what price-back accepts."""
    try:
        conn = db.connect()
        db.migrate(conn)
        try:
            saved = storage.save_to_postgres(
                conn,
                store_key=market_name.lower().replace(" ", "-"),
                store_name=market_name,
                produtos=produtos,
                source="admin_link",
                search_term=url,
            )
            logger.info("[%s] %d produtos salvos na coleta bruta (price_bot_db).", market_name, saved)
        finally:
            conn.close()
    except Exception:
        logger.warning("Não foi possível salvar a coleta bruta no Postgres.", exc_info=True)


def register_market(payload: CollectRequest, client: httpx.Client | None = None) -> None:
    """Registers (or updates) the market in price-back with its storefront
    link and address, ahead of any scraping. Best-effort and never raises:
    a market's address is worth saving even if the site turns out to be
    unscrapable, and a previous successful registration must not be lost
    just because this call fails.

    price-back keys a market by name, not by URL, and accumulates every
    distinct link submitted for that name — so a chain with more than one
    domain (or a market whose site moved) just ends up with several links
    instead of several markets. This is also what the Admin panel's
    "Recarregar" button re-collects: every link stored for a market.
    """
    owns_client = client is None
    if client is None:
        client = httpx.Client(timeout=15)

    try:
        resp = client.post(
            f"{PRICE_BACK_URL}/api/markets",
            json={
                "name": payload.market_name,
                "source": "bot",
                "link": payload.url,
                "cep": payload.cep,
                "street": payload.street,
                "number": payload.number,
                "neighborhood": payload.neighborhood,
                "city": payload.city,
                "state": payload.state,
                "complement": payload.complement,
            },
        )
        if resp.status_code == 201:
            logger.info(
                "[%s] Mercado registrado em price-back (id=%s, endereço=%s).",
                payload.market_name,
                resp.json().get("id"),
                "sim" if (payload.cep or payload.street or payload.city) else "não informado",
            )
        else:
            logger.warning(
                "price-back rejeitou o registro do mercado '%s': %s %s",
                payload.market_name,
                resp.status_code,
                resp.text,
            )
    except httpx.HTTPError:
        logger.warning(
            "Falha ao registrar o mercado '%s' no price-back.", payload.market_name, exc_info=True
        )
    finally:
        if owns_client:
            client.close()


def push_to_price_back(
    market_name: str,
    produtos_validos: list[dict],
    client: httpx.Client | None = None,
    collected_at: str | None = None,
) -> list[dict]:
    """Sends each AI-classified product to price-back's
    POST /api/products, tagged source=bot, together with what
    nextprice.product_parser extracts from its name (brand, package size,
    generic key...) and the time the price was collected. Returns the ones
    price-back accepted (as the JSON it returned).

    Accepts an injected `client` for tests (e.g. one built with
    httpx.MockTransport); a real client is opened/closed automatically
    otherwise.
    """
    owns_client = client is None
    if client is None:
        client = httpx.Client(timeout=15)

    inserted: list[dict] = []
    try:
        for item in produtos_validos:
            try:
                body = {
                    "name": item["nome"],
                    "unit": item.get("unidade") or "un",
                    "category": item.get("categoria") or "OUTROS",
                    "price": item["preco_numerico"],
                    "market_name": market_name,
                    "source": "bot",
                    **price_back_fields(parse_product_name(item["nome"])),
                }
                if collected_at:
                    body["collected_at"] = collected_at
                resp = client.post(f"{PRICE_BACK_URL}/api/products", json=body)
                if resp.status_code == 201:
                    inserted.append(resp.json())
                else:
                    logger.warning(
                        "price-back rejeitou '%s': %s %s",
                        item.get("nome"),
                        resp.status_code,
                        resp.text,
                    )
            except httpx.HTTPError:
                logger.warning(
                    "Falha ao chamar price-back para '%s'.", item.get("nome"), exc_info=True
                )
    finally:
        if owns_client:
            client.close()

    logger.info(
        "[%s] %d/%d produtos aceitos por price-back.", market_name, len(inserted), len(produtos_validos)
    )
    return inserted


@app.post("/collect", response_model=CollectResponse)
def collect(payload: CollectRequest) -> CollectResponse:
    logger.info("[%s] Coleta solicitada — url=%s", payload.market_name, payload.url)
    register_market(payload)

    store = find_store_by_url(payload.url)

    if store is not None:
        logger.info(
            "[%s] Loja conhecida (%s) — usando seletores CSS registrados.", payload.market_name, store.key
        )
        try:
            produtos_brutos = scrape_known_store(payload.url, store)
        except Exception as exc:
            logger.error("[%s] Falha ao acessar a URL: %s", payload.market_name, exc, exc_info=True)
            raise HTTPException(status_code=502, detail=f"Falha ao acessar a URL: {exc}") from exc
        blocks_found = len(produtos_brutos)
        logger.info("[%s] %d produtos encontrados na página.", payload.market_name, blocks_found)
        produtos = classify_products(payload.market_name, produtos_brutos)
        logger.info("[%s] %d produtos classificados (categoria/unidade).", payload.market_name, len(produtos))
    else:
        logger.info(
            "[%s] Loja não registrada — usando extração genérica via IA.", payload.market_name
        )
        try:
            blocks = scrape_blocks(payload.url)
        except Exception as exc:
            logger.error("[%s] Falha ao acessar a URL: %s", payload.market_name, exc, exc_info=True)
            raise HTTPException(status_code=502, detail=f"Falha ao acessar a URL: {exc}") from exc
        blocks_found = len(blocks)
        logger.info("[%s] %d blocos de texto coletados da página.", payload.market_name, blocks_found)
        ai_client = GeminiClient()
        produtos = ai_client.extract_products_generic(payload.market_name, blocks)
        logger.info(
            "[%s] IA reconheceu %d produtos a partir dos blocos.", payload.market_name, len(produtos)
        )

    # Vitrine banners ("↪️ Atacado SUPER!") come through the same selectors and,
    # for unknown stores, through the AI too — never register them as products.
    reais = [p for p in produtos if not looks_like_banner(p.get("nome", ""))]
    if len(reais) < len(produtos):
        logger.info(
            "[%s] %d banners/rótulos de vitrine descartados.",
            payload.market_name,
            len(produtos) - len(reais),
        )
    produtos = reais

    validos = []
    for item in produtos:
        try:
            item["preco_numerico"] = parse_price_brl(item["preco"])
        except (ValueError, KeyError, TypeError):
            continue
        validos.append(item)
    if len(validos) < len(produtos):
        logger.info(
            "[%s] %d produtos descartados por preço inválido/ilegível.",
            payload.market_name,
            len(produtos) - len(validos),
        )

    save_raw_scrape(payload.market_name, payload.url, produtos)
    collected_at = datetime.now(timezone.utc).isoformat()
    inserted = push_to_price_back(payload.market_name, validos, collected_at=collected_at)

    logger.info(
        "[%s] Coleta concluída: %d encontrados, %d extraídos, %d inseridos em price-back.",
        payload.market_name,
        blocks_found,
        len(produtos),
        len(inserted),
    )

    return CollectResponse(
        market_name=payload.market_name,
        blocks_found=blocks_found,
        products_extracted=len(produtos),
        products_inserted=len(inserted),
        products=[
            CollectedProduct(
                name=p["name"], price=p["default_price"], category=p["category"], unit=p["unit"]
            )
            for p in inserted
        ],
    )


def main() -> None:
    import uvicorn

    port = int(os.environ.get("COLLECT_PORT", "8085"))
    uvicorn.run(app, host="0.0.0.0", port=port)


if __name__ == "__main__":
    main()
