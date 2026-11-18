from __future__ import annotations

import json
from datetime import datetime

import httpx
import pytest
from fastapi.testclient import TestClient

from nextprice import collect_api

# Saved before the autouse fixture below patches collect_api.register_market,
# so the tests that exercise register_market itself call the real thing.
_real_register_market = collect_api.register_market


@pytest.fixture(autouse=True)
def _stub_register_market(monkeypatch: pytest.MonkeyPatch) -> list:
    """collect() registers the market with price-back before scraping —
    stub it out for every test in this file so none of them make a real
    network call; tests that care about the payload replace this stub or
    call register_market directly (see test_register_market_*)."""
    calls: list = []
    monkeypatch.setattr(collect_api, "register_market", lambda payload, client=None: calls.append(payload))
    return calls


def test_push_to_price_back_returns_only_accepted_items() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        body = request.read()
        if b"Aceito" in body:
            return httpx.Response(
                201,
                json={"name": "Aceito", "unit": "un", "category": "OUTROS", "default_price": 5.0},
            )
        return httpx.Response(400, json={"error": "invalid"})

    client = httpx.Client(transport=httpx.MockTransport(handler))

    produtos = [
        {"nome": "Aceito", "preco_numerico": 5.0, "categoria": "OUTROS", "unidade": "un"},
        {"nome": "Rejeitado", "preco_numerico": 3.0, "categoria": "OUTROS", "unidade": "un"},
    ]

    inserted = collect_api.push_to_price_back("Loja Teste", produtos, client=client)

    assert len(inserted) == 1
    assert inserted[0]["name"] == "Aceito"


def test_push_to_price_back_network_error_is_swallowed() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        raise httpx.ConnectError("connection refused")

    client = httpx.Client(transport=httpx.MockTransport(handler))

    inserted = collect_api.push_to_price_back(
        "Loja Teste", [{"nome": "X", "preco_numerico": 1.0}], client=client
    )

    assert inserted == []


def test_push_to_price_back_empty_list_returns_empty() -> None:
    assert collect_api.push_to_price_back("Loja", []) == []


def test_register_market_sends_url_as_link_and_address_fields() -> None:
    captured: dict = {}

    def handler(request: httpx.Request) -> httpx.Response:
        captured["json"] = json.loads(request.read())
        return httpx.Response(201, json={"id": "m1"})

    client = httpx.Client(transport=httpx.MockTransport(handler))
    payload = collect_api.CollectRequest(
        url="https://loja.exemplo/",
        market_name="Loja Exemplo",
        cep="64000-000",
        street="Rua X",
        number="10",
        city="Teresina",
        state="PI",
    )

    _real_register_market(payload, client=client)

    assert captured["json"] == {
        "name": "Loja Exemplo",
        "source": "bot",
        "link": "https://loja.exemplo/",
        "cep": "64000-000",
        "street": "Rua X",
        "number": "10",
        "neighborhood": "",
        "city": "Teresina",
        "state": "PI",
        "complement": "",
    }


def test_register_market_network_error_is_swallowed() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        raise httpx.ConnectError("connection refused")

    client = httpx.Client(transport=httpx.MockTransport(handler))
    payload = collect_api.CollectRequest(url="https://loja.exemplo/", market_name="Loja Exemplo")

    _real_register_market(payload, client=client)  # must not raise


def test_register_market_rejection_is_swallowed() -> None:
    client = httpx.Client(
        transport=httpx.MockTransport(lambda req: httpx.Response(400, json={"error": "bad"}))
    )
    payload = collect_api.CollectRequest(url="https://loja.exemplo/", market_name="Loja Exemplo")

    _real_register_market(payload, client=client)  # must not raise


@pytest.fixture
def client() -> TestClient:
    return TestClient(collect_api.app)


class _FakeGeminiClient:
    """Stands in for GeminiClient so tests never construct the real thing
    (which loads GEMINI_API_KEY via Settings.load() and would fail in a
    fresh checkout without a .env file)."""

    def __init__(self, produtos: list[dict]) -> None:
        self._produtos = produtos

    def extract_products_generic(self, loja: str, blocks: list[str]) -> list[dict]:
        return self._produtos


def test_collect_endpoint_happy_path(monkeypatch: pytest.MonkeyPatch, client: TestClient) -> None:
    monkeypatch.setattr(collect_api, "scrape_blocks", lambda url: ["Tomate 1kg - R$ 7,79"])
    monkeypatch.setattr(
        collect_api,
        "GeminiClient",
        lambda: _FakeGeminiClient(
            [{"nome": "Tomate 1kg", "preco": "R$ 7,79", "categoria": "HORTIFRUTI", "unidade": "kg"}]
        ),
    )
    monkeypatch.setattr(collect_api, "save_raw_scrape", lambda *a, **k: None)
    monkeypatch.setattr(
        collect_api,
        "push_to_price_back",
        lambda market_name, validos, **kwargs: [
            {"name": "Tomate 1kg", "unit": "kg", "category": "HORTIFRUTI", "default_price": 7.79}
        ],
    )

    resp = client.post(
        "/collect", json={"url": "https://loja.exemplo/", "market_name": "Loja Exemplo"}
    )

    assert resp.status_code == 200
    data = resp.json()
    assert data["market_name"] == "Loja Exemplo"
    assert data["blocks_found"] == 1
    assert data["products_extracted"] == 1
    assert data["products_inserted"] == 1
    assert data["products"] == [
        {"name": "Tomate 1kg", "price": 7.79, "category": "HORTIFRUTI", "unit": "kg"}
    ]


def test_collect_endpoint_filters_unparseable_prices_before_pushing(
    monkeypatch: pytest.MonkeyPatch, client: TestClient
) -> None:
    monkeypatch.setattr(collect_api, "scrape_blocks", lambda url: ["bloco"])
    monkeypatch.setattr(
        collect_api,
        "GeminiClient",
        lambda: _FakeGeminiClient(
            [
                {"nome": "Sem preço", "preco": "sob consulta"},
                {"nome": "Com preço", "preco": "R$ 4,00"},
            ]
        ),
    )
    monkeypatch.setattr(collect_api, "save_raw_scrape", lambda *a, **k: None)

    captured: dict = {}

    def fake_push(market_name: str, validos: list[dict], **kwargs) -> list[dict]:
        captured["validos"] = validos
        captured["kwargs"] = kwargs
        return []

    monkeypatch.setattr(collect_api, "push_to_price_back", fake_push)

    resp = client.post(
        "/collect", json={"url": "https://loja.exemplo/", "market_name": "Loja Exemplo"}
    )

    assert resp.status_code == 200
    assert len(captured["validos"]) == 1
    assert captured["validos"][0]["nome"] == "Com preço"


def test_collect_endpoint_scrape_failure_returns_502(
    monkeypatch: pytest.MonkeyPatch, client: TestClient
) -> None:
    def boom(url: str) -> list[str]:
        raise RuntimeError("site fora do ar")

    monkeypatch.setattr(collect_api, "scrape_blocks", boom)

    resp = client.post(
        "/collect", json={"url": "https://loja.exemplo/", "market_name": "Loja Exemplo"}
    )

    assert resp.status_code == 502


def test_collect_endpoint_requires_market_name(client: TestClient) -> None:
    resp = client.post("/collect", json={"url": "https://loja.exemplo/", "market_name": ""})
    assert resp.status_code == 422


def test_collect_endpoint_registers_market_even_when_scrape_fails(
    monkeypatch: pytest.MonkeyPatch, client: TestClient, _stub_register_market: list
) -> None:
    """The market's address is worth saving even if the site turns out to
    be unscrapable — registration must happen before the scrape attempt,
    not after."""
    monkeypatch.setattr(
        collect_api, "scrape_blocks", lambda url: (_ for _ in ()).throw(RuntimeError("boom"))
    )

    resp = client.post(
        "/collect",
        json={
            "url": "https://loja.exemplo/",
            "market_name": "Loja Exemplo",
            "cep": "64000-000",
            "city": "Teresina",
            "state": "PI",
        },
    )

    assert resp.status_code == 502
    assert len(_stub_register_market) == 1
    assert _stub_register_market[0].market_name == "Loja Exemplo"
    assert _stub_register_market[0].cep == "64000-000"


class _FakeGeminiClassifier:
    """Stands in for GeminiClient in the known-store path (classify_categories
    only — unlike _FakeGeminiClient, this path never calls extract_products_generic)."""

    def __init__(self, classificacoes: dict[str, dict]) -> None:
        self._classificacoes = classificacoes

    def classify_categories(self, loja: str, nomes_produtos: list[str]) -> dict[str, dict]:
        return self._classificacoes


def test_classify_products_empty_list_returns_empty() -> None:
    assert collect_api.classify_products("Loja", []) == []


def test_classify_products_defaults_when_ai_returns_nothing(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    monkeypatch.setattr(collect_api, "GeminiClient", lambda: _FakeGeminiClassifier({}))

    produtos = collect_api.classify_products("Loja", [{"nome": "Item X", "preco": "R$ 1,00"}])

    assert produtos == [
        {
            "nome": "Item X",
            "preco": "R$ 1,00",
            "categoria": "OUTROS",
            "unidade": "un",
            "loja": "Loja",
        }
    ]


def test_classify_products_uses_ai_classification_when_available(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    monkeypatch.setattr(
        collect_api,
        "GeminiClient",
        lambda: _FakeGeminiClassifier(
            {"Tomate 1kg": {"categoria": "HORTIFRUTI", "unidade": "kg"}}
        ),
    )

    produtos = collect_api.classify_products("Loja", [{"nome": "Tomate 1kg", "preco": "R$ 7,79"}])

    assert produtos[0]["categoria"] == "HORTIFRUTI"
    assert produtos[0]["unidade"] == "kg"


def test_collect_endpoint_known_store_uses_css_selectors_not_ai_for_name_price(
    monkeypatch: pytest.MonkeyPatch, client: TestClient
) -> None:
    monkeypatch.setattr(
        collect_api,
        "scrape_known_store",
        lambda url, store: [{"nome": "Tomate 1kg", "preco": "R$ 7,79"}],
    )
    monkeypatch.setattr(
        collect_api,
        "GeminiClient",
        lambda: _FakeGeminiClassifier(
            {"Tomate 1kg": {"categoria": "HORTIFRUTI", "unidade": "kg"}}
        ),
    )
    monkeypatch.setattr(collect_api, "save_raw_scrape", lambda *a, **k: None)
    monkeypatch.setattr(
        collect_api,
        "push_to_price_back",
        lambda market_name, validos, **kwargs: [
            {"name": "Tomate 1kg", "unit": "kg", "category": "HORTIFRUTI", "default_price": 7.79}
        ],
    )

    resp = client.post(
        "/collect",
        json={
            "url": "https://www.carvalhosupershop.com.br/",
            "market_name": "CarvalhoSuperShop",
        },
    )

    assert resp.status_code == 200
    data = resp.json()
    assert data["blocks_found"] == 1
    assert data["products_extracted"] == 1
    assert data["products_inserted"] == 1


def test_collect_endpoint_known_store_survives_ai_classification_failure(
    monkeypatch: pytest.MonkeyPatch, client: TestClient
) -> None:
    """Core resilience guarantee: for a registered store, real nome/preco
    still reaches price-back even when the AI is completely unavailable —
    unlike the generic (unregistered-market) flow, which drops everything
    if the AI fails. Falls back to the keyword classifier (nextprice.categorize),
    not a blind "OUTROS"/"un", so "Tomate 1kg" still lands correctly."""
    monkeypatch.setattr(
        collect_api,
        "scrape_known_store",
        lambda url, store: [{"nome": "Tomate 1kg", "preco": "R$ 7,79"}],
    )
    monkeypatch.setattr(collect_api, "GeminiClient", lambda: _FakeGeminiClassifier({}))
    monkeypatch.setattr(collect_api, "save_raw_scrape", lambda *a, **k: None)

    captured: dict = {}

    def fake_push(market_name: str, validos: list[dict], **kwargs) -> list[dict]:
        captured["validos"] = validos
        captured["kwargs"] = kwargs
        return []

    monkeypatch.setattr(collect_api, "push_to_price_back", fake_push)

    resp = client.post(
        "/collect",
        json={
            "url": "https://www.carvalhosupershop.com.br/",
            "market_name": "CarvalhoSuperShop",
        },
    )

    assert resp.status_code == 200
    assert len(captured["validos"]) == 1
    produto = captured["validos"][0]
    assert produto["nome"] == "Tomate 1kg"
    assert produto["categoria"] == "HORTIFRUTI"
    assert produto["unidade"] == "kg"


def test_collect_endpoint_known_store_scrape_failure_returns_502(
    monkeypatch: pytest.MonkeyPatch, client: TestClient
) -> None:
    def boom(url: str, store: object) -> list[dict]:
        raise RuntimeError("site fora do ar")

    monkeypatch.setattr(collect_api, "scrape_known_store", boom)

    resp = client.post(
        "/collect",
        json={
            "url": "https://www.carvalhosupershop.com.br/",
            "market_name": "CarvalhoSuperShop",
        },
    )

    assert resp.status_code == 502


def _capturing_client(bodies: list[dict]) -> httpx.Client:
    def handler(request: httpx.Request) -> httpx.Response:
        body = json.loads(request.read())
        bodies.append(body)
        return httpx.Response(201, json={"name": body["name"], "default_price": body["price"]})

    return httpx.Client(transport=httpx.MockTransport(handler))


def test_push_to_price_back_sends_parsed_fields_and_collection_time() -> None:
    bodies: list[dict] = []
    produtos = [
        {
            "nome": "Café Melitta Tradicional Vácuo 500g",
            "preco_numerico": 28.9,
            "categoria": "GRÃOS",
            "unidade": "un",
        }
    ]

    collect_api.push_to_price_back(
        "Loja", produtos, client=_capturing_client(bodies), collected_at="2026-09-19T10:00:00+00:00"
    )

    (body,) = bodies
    assert body["name"] == "Café Melitta Tradicional Vácuo 500g"
    assert body["brand"] == "Melitta"
    assert body["size_kind"] == "mass"
    assert body["size_amount"] == 500
    assert body["size_label"] == "500 g"
    assert body["pack_count"] == 1
    assert body["generic_key"] == "cafe tradicional vacuo"
    assert body["collected_at"] == "2026-09-19T10:00:00+00:00"
    assert body["source"] == "bot"


def test_push_to_price_back_omits_size_when_name_has_none() -> None:
    bodies: list[dict] = []
    produtos = [{"nome": "Abacaxi Premium", "preco_numerico": 5.0, "categoria": "HORTIFRUTI"}]

    collect_api.push_to_price_back("Loja", produtos, client=_capturing_client(bodies))

    (body,) = bodies
    assert "size_kind" not in body and "size_amount" not in body
    assert "collected_at" not in body
    assert body["brand"] == ""


def test_collect_endpoint_drops_banners_and_stamps_collection_time(
    monkeypatch: pytest.MonkeyPatch, client: TestClient
) -> None:
    monkeypatch.setattr(collect_api, "scrape_blocks", lambda url: ["bloco"])
    monkeypatch.setattr(
        collect_api,
        "GeminiClient",
        lambda: _FakeGeminiClient(
            [
                {"nome": "↪️ Atacado SUPER!", "preco": "R$ 7,99", "categoria": "OUTROS"},
                {"nome": "Tomate 1kg", "preco": "R$ 7,79", "categoria": "HORTIFRUTI"},
            ]
        ),
    )
    monkeypatch.setattr(collect_api, "save_raw_scrape", lambda *a, **k: None)

    captured: dict = {}

    def fake_push(market_name: str, validos: list[dict], **kwargs) -> list[dict]:
        captured["validos"] = validos
        captured["kwargs"] = kwargs
        return []

    monkeypatch.setattr(collect_api, "push_to_price_back", fake_push)

    resp = client.post(
        "/collect", json={"url": "https://loja.exemplo/", "market_name": "Loja Exemplo"}
    )

    assert resp.status_code == 200
    assert [p["nome"] for p in captured["validos"]] == ["Tomate 1kg"]
    assert resp.json()["products_extracted"] == 1
    collected_at = captured["kwargs"]["collected_at"]
    assert datetime.fromisoformat(collected_at).tzinfo is not None
