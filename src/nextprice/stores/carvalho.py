from nextprice.stores.base import StoreConfig

STORE = StoreConfig(
    key="carvalhosupershop",
    name="CarvalhoSuperShop",
    base_url="https://www.carvalhosupershop.com.br/",
    card_selector="div.ng-star-inserted",
    price_selector="[data-cy='preco']",
)
