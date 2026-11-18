from urllib.parse import urlparse

from nextprice.stores import carvalho, ferreira
from nextprice.stores.base import StoreConfig

STORES: list[StoreConfig] = [carvalho.STORE, ferreira.STORE]


def _host(url: str) -> str:
    host = urlparse(url).netloc.lower()
    return host.removeprefix("www.")


def find_store_by_url(url: str) -> StoreConfig | None:
    """Matches a URL to a registered store by domain, so the admin panel's
    on-demand collection can reuse a store's known CSS selectors (precise
    nome/preco, no AI needed) instead of the generic AI-only heuristic used
    for unregistered markets."""
    host = _host(url)
    if not host:
        return None
    return next((store for store in STORES if _host(store.base_url) == host), None)
