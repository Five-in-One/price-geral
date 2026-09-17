from __future__ import annotations

import logging

import psycopg
from playwright.sync_api import sync_playwright

from nextprice import db
from nextprice.browser import scrape_listing, scroll_to_load
from nextprice.storage import dedupe_by_name, save_json, save_to_postgres
from nextprice.stores.base import StoreConfig
from nextprice.stores.registry import STORES

logging.basicConfig(level=logging.INFO, format="%(message)s")
logger = logging.getLogger(__name__)


def run_store(store: StoreConfig, conn: psycopg.Connection | None) -> None:
    logger.info("[%s] Acessando: %s", store.name, store.base_url)
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        page = browser.new_page()
        page.goto(store.base_url, timeout=60000)
        page.wait_for_timeout(5000)
        scroll_to_load(page)
        produtos = scrape_listing(page, store)
        browser.close()

    unicos = dedupe_by_name(produtos)
    save_json(store.name, unicos)

    if conn is not None:
        save_to_postgres(conn, store.key, store.name, unicos, source="extract")


def _connect_or_none() -> psycopg.Connection | None:
    """Best-effort Postgres connection: the extract flow must keep working
    (JSON output only) even if the database isn't up."""
    try:
        conn = db.connect()
        db.migrate(conn)
        return conn
    except psycopg.OperationalError:
        logger.warning(
            "Postgres indisponível — salvando apenas em JSON "
            "(rode `docker compose up -d` no price-back para persistir também no banco)."
        )
        return None


def main() -> None:
    conn = _connect_or_none()
    try:
        for store in STORES:
            run_store(store, conn)
    finally:
        if conn is not None:
            conn.close()


if __name__ == "__main__":
    main()
