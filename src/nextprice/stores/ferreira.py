from nextprice.stores.base import StoreConfig

STORE = StoreConfig(
    key="ferreiraemcasa",
    name="FerreiraEmCasa",
    base_url="https://www.ferreiraemcasa.com.br/",
    card_selector="div.ng-star-inserted:has(span.font-bold)",
    price_selector="span.font-bold",
    exclude_name_substrings=("faça seu login", "departamentos"),
)
