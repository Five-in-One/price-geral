from __future__ import annotations

import httpx
import pytest
from fastapi.testclient import TestClient

from nextprice import collect_api


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
        lambda market_name, validos: [
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

    def fake_push(market_name: str, validos: list[dict]) -> list[dict]:
        captured["validos"] = validos
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
