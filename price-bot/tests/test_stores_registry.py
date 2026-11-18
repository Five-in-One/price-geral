from nextprice.stores.registry import STORES, find_store_by_url


def test_stores_nao_vazio() -> None:
    assert len(STORES) >= 1


def test_stores_chaves_unicas() -> None:
    chaves = [store.key for store in STORES]
    assert len(chaves) == len(set(chaves))


def test_stores_campos_obrigatorios_preenchidos() -> None:
    for store in STORES:
        assert store.key
        assert store.name
        assert store.base_url.startswith("http")
        assert store.card_selector
        assert store.price_selector


def test_find_store_by_url_matches_registered_domain() -> None:
    store = find_store_by_url("https://www.carvalhosupershop.com.br/")
    assert store is not None
    assert store.key == "carvalhosupershop"


def test_find_store_by_url_ignores_www_and_path() -> None:
    store = find_store_by_url("https://carvalhosupershop.com.br/ofertas?x=1")
    assert store is not None
    assert store.key == "carvalhosupershop"


def test_find_store_by_url_unregistered_domain_returns_none() -> None:
    assert find_store_by_url("https://www.mercado-desconhecido.com.br/") is None


def test_find_store_by_url_invalid_url_returns_none() -> None:
    assert find_store_by_url("") is None
