from __future__ import annotations

from dataclasses import dataclass, field

_DEFAULT_NAME_SELECTOR = "h2, h3, .product-title, span[class*='title'], a"
_DEFAULT_SEARCH_CARD_SELECTOR = (
    "div.ng-star-inserted:has-text('R$'), article, .product-item, div[class*='product']"
)


@dataclass(frozen=True)
class StoreConfig:
    """Configuração declarativa de uma loja suportada.

    Adicionar uma nova loja = criar um novo StoreConfig e registrá-lo em
    stores/registry.py — não é necessário duplicar lógica de scraping.
    """

    key: str
    name: str
    base_url: str
    card_selector: str
    price_selector: str
    name_selector: str = _DEFAULT_NAME_SELECTOR
    require_price_prefix: str | None = "R$"
    exclude_name_substrings: tuple[str, ...] = field(default_factory=tuple)
    search_input_selector: str = "#search-term"
    search_card_selector: str = _DEFAULT_SEARCH_CARD_SELECTOR
