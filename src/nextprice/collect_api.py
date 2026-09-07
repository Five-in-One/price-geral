"""HTTP API used by the Admin panel's "coleta sob demanda": given a market
URL, scrape it, let the AI classify the products found, save the raw
result, and push the classified products into price-back.

Run with: `uvicorn nextprice.collect_api:app --port 8085` (or
`python -m nextprice.collect_api`).
"""

from __future__ import annotations

import logging
import os

import httpx
from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from playwright.sync_api import sync_playwright
from pydantic import BaseModel, Field

from nextprice import db, storage
from nextprice.ai.client import GeminiClient
from nextprice.browser import collect_listing_text_blocks
from nextprice.pricing import parse_price_brl

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


def save_raw_scrape(market_name: str, url: str, produtos: list[dict]) -> None:
    """Best-effort: keeps the raw AI output in price-bot's own staging
    table regardless of what price-back accepts."""
    try:
        conn = db.connect()
        db.migrate(conn)
        try:
            storage.save_to_postgres(
                conn,
                store_key=market_name.lower().replace(" ", "-"),
                store_name=market_name,
                produtos=produtos,
                source="admin_link",
                search_term=url,
            )
        finally:
            conn.close()
    except Exception:
        logger.warning("Não foi possível salvar a coleta bruta no Postgres.", exc_info=True)


def push_to_price_back(
    market_name: str, produtos_validos: list[dict], client: httpx.Client | None = None
) -> list[dict]:
    """Sends each AI-classified product to price-back's
    POST /api/products, tagged source=bot. Returns the ones price-back
    accepted (as the JSON it returned).

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
                resp = client.post(
                    f"{PRICE_BACK_URL}/api/products",
                    json={
                        "name": item["nome"],
                        "unit": item.get("unidade") or "un",
                        "category": item.get("categoria") or "OUTROS",
                        "price": item["preco_numerico"],
                        "market_name": market_name,
                        "source": "bot",
                    },
                )
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

    return inserted


@app.post("/collect", response_model=CollectResponse)
def collect(payload: CollectRequest) -> CollectResponse:
    try:
        blocks = scrape_blocks(payload.url)
    except Exception as exc:
        raise HTTPException(status_code=502, detail=f"Falha ao acessar a URL: {exc}") from exc

    ai_client = GeminiClient()
    produtos = ai_client.extract_products_generic(payload.market_name, blocks)

    validos = []
    for item in produtos:
        try:
            item["preco_numerico"] = parse_price_brl(item["preco"])
        except (ValueError, KeyError, TypeError):
            continue
        validos.append(item)

    save_raw_scrape(payload.market_name, payload.url, produtos)
    inserted = push_to_price_back(payload.market_name, validos)

    return CollectResponse(
        market_name=payload.market_name,
        blocks_found=len(blocks),
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
