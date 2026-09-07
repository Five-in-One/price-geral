from __future__ import annotations

import os

import psycopg
import pytest

from nextprice import db, storage

TEST_DATABASE_URL = os.environ.get(
    "TEST_DATABASE_URL",
    "postgresql://postgres:postgres@localhost:5432/price_bot_db_test",
)


@pytest.fixture
def conn() -> psycopg.Connection:
    """Connects to the real dockerized test database, migrates it, and
    truncates scraped_products so every test starts from a clean slate.

    Skips (rather than fails) if Postgres isn't reachable, so the rest of
    the suite still runs in environments without `docker compose up -d`.
    """
    try:
        connection = db.connect(TEST_DATABASE_URL)
    except psycopg.OperationalError as exc:
        pytest.skip(f"Postgres indisponível em {TEST_DATABASE_URL}: {exc}")

    db.migrate(connection)
    with connection.cursor() as cur:
        cur.execute("TRUNCATE TABLE scraped_products RESTART IDENTITY")
    connection.commit()

    yield connection
    connection.close()


def _fetch_all_rows(conn: psycopg.Connection) -> list[tuple]:
    with conn.cursor() as cur:
        cur.execute(
            """
            SELECT store_key, store_name, product_name, price_text, price, source, search_term
            FROM scraped_products
            ORDER BY id
            """
        )
        return cur.fetchall()


def test_save_to_postgres_inserts_parsed_price(conn: psycopg.Connection) -> None:
    produtos = [{"nome": "Tomate", "preco": "R$ 7,79", "data_coleta": "2026-01-01 10:00:00"}]

    inserted = storage.save_to_postgres(
        conn, "carvalhosupershop", "CarvalhoSuperShop", produtos, source="extract"
    )

    assert inserted == 1
    rows = _fetch_all_rows(conn)
    assert len(rows) == 1
    store_key, store_name, product_name, price_text, price, source, search_term = rows[0]
    assert (store_key, store_name, product_name, price_text, source, search_term) == (
        "carvalhosupershop",
        "CarvalhoSuperShop",
        "Tomate",
        "R$ 7,79",
        "extract",
        None,
    )
    assert float(price) == 7.79


def test_save_to_postgres_track_source_stores_search_term(conn: psycopg.Connection) -> None:
    produtos = [{"nome": "Tomate Longa Vida 1kg", "preco": "R$ 8,49", "loja": "FerreiraEmCasa"}]

    inserted = storage.save_to_postgres(
        conn,
        "ferreiraemcasa",
        "FerreiraEmCasa",
        produtos,
        source="track",
        search_term="tomate",
    )

    assert inserted == 1
    rows = _fetch_all_rows(conn)
    assert rows[0][5:] == ("track", "tomate")


def test_save_to_postgres_unparseable_price_keeps_row_with_null_price(
    conn: psycopg.Connection,
) -> None:
    produtos = [{"nome": "Produto sob encomenda", "preco": "sob consulta"}]

    inserted = storage.save_to_postgres(
        conn, "carvalhosupershop", "CarvalhoSuperShop", produtos, source="extract"
    )

    assert inserted == 1
    rows = _fetch_all_rows(conn)
    assert rows[0][3] == "sob consulta"
    assert rows[0][4] is None


def test_save_to_postgres_skips_items_missing_name_or_price(conn: psycopg.Connection) -> None:
    produtos = [
        {"nome": "Sem preço"},
        {"preco": "R$ 1,00"},
        {"nome": "Válido", "preco": "R$ 2,00"},
    ]

    inserted = storage.save_to_postgres(
        conn, "carvalhosupershop", "CarvalhoSuperShop", produtos, source="extract"
    )

    assert inserted == 1
    rows = _fetch_all_rows(conn)
    assert rows[0][2] == "Válido"


def test_save_to_postgres_empty_list_returns_zero(conn: psycopg.Connection) -> None:
    assert storage.save_to_postgres(conn, "k", "Nome", [], source="extract") == 0
    assert _fetch_all_rows(conn) == []


def test_migrate_is_idempotent(conn: psycopg.Connection) -> None:
    with conn.cursor() as cur:
        cur.execute("SELECT COUNT(*) FROM schema_migrations")
        count_before = cur.fetchone()[0]

    db.migrate(conn)  # fixture already migrated once; must not raise or duplicate

    with conn.cursor() as cur:
        cur.execute("SELECT COUNT(*) FROM schema_migrations")
        count_after = cur.fetchone()[0]

    assert count_after == count_before
