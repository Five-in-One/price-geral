from __future__ import annotations

import logging
import os
from pathlib import Path

import psycopg
from dotenv import find_dotenv, load_dotenv

logger = logging.getLogger(__name__)

_DEFAULT_DATABASE_URL = "postgresql://postgres:postgres@localhost:5432/price_bot_db"

# migrations/ lives at the repo root, two levels above this file
# (src/nextprice/db.py -> src/nextprice -> src -> repo root).
_MIGRATIONS_DIR = Path(__file__).resolve().parents[2] / "migrations"


def get_database_url() -> str:
    """Reads DATABASE_URL from the environment, loading .env if present.

    Kept independent from ai.client.Settings on purpose: the extract flow
    persists to Postgres too, and must keep working without a
    GEMINI_API_KEY configured.
    """
    load_dotenv(find_dotenv(usecwd=True))
    return os.environ.get("DATABASE_URL", _DEFAULT_DATABASE_URL).strip()


def connect(database_url: str | None = None, connect_timeout: int = 5) -> psycopg.Connection:
    """Opens a new connection to the bot's Postgres database.

    A short connect_timeout keeps the best-effort callers (extractor/tracker
    CLIs) from hanging for the OS's default TCP timeout when Postgres isn't
    reachable.
    """
    return psycopg.connect(database_url or get_database_url(), connect_timeout=connect_timeout)


def migrate(conn: psycopg.Connection) -> None:
    """Applies every migrations/*.sql file not yet recorded in
    schema_migrations, in filename order."""
    with conn.cursor() as cur:
        cur.execute(
            """
            CREATE TABLE IF NOT EXISTS schema_migrations (
                version     text PRIMARY KEY,
                applied_at  timestamptz NOT NULL DEFAULT now()
            )
            """
        )
        conn.commit()

        cur.execute("SELECT version FROM schema_migrations")
        applied = {row[0] for row in cur.fetchall()}

        for path in sorted(_MIGRATIONS_DIR.glob("*.sql")):
            if path.name in applied:
                continue

            sql = path.read_text(encoding="utf-8")
            cur.execute(sql)
            cur.execute("INSERT INTO schema_migrations (version) VALUES (%s)", (path.name,))
            conn.commit()
            logger.info("Migration aplicada: %s", path.name)
