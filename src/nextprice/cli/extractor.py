from __future__ import annotations

import logging

from playwright.sync_api import sync_playwright

from nextprice.browser import scrape_listing, scroll_to_load
from nextprice.storage import save_json
from nextprice.stores.base import StoreConfig
from nextprice.stores.registry import STORES

logging.basicConfig(level=logging.INFO, format="%(message)s")
logger = logging.getLogger(__name__)


def run_store(store: StoreConfig) -> None:
    logger.info("[%s] Acessando: %s", store.name, store.base_url)
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        page = browser.new_page()
        page.goto(store.base_url, timeout=60000)
        page.wait_for_timeout(5000)
        scroll_to_load(page)
        produtos = scrape_listing(page, store)
        browser.close()
    save_json(store.name, produtos)


def main() -> None:
    for store in STORES:
        run_store(store)


if __name__ == "__main__":
    main()
