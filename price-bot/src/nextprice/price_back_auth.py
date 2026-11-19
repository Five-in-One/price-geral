"""Bearer token for the bot's calls to price-back.

price-back is a Resource Server: writing markets and products needs a JWT
issued by price-auth (the Authorization Server). The bot logs in with its own
account (BOT_USERNAME / BOT_PASSWORD, an ``admin`` account), keeps the token
in memory and logs in again shortly before it expires or when price-back
answers 401.

Without BOT_USERNAME the bot sends no Authorization header, which is what the
unit tests (with mocked HTTP) rely on.
"""

from __future__ import annotations

import base64
import json
import logging
import os
import threading
import time

import httpx

logger = logging.getLogger(__name__)

PRICE_AUTH_URL = os.environ.get("PRICE_AUTH_URL", "http://localhost:8081").rstrip("/")

_lock = threading.Lock()
_token: str | None = None
_expires_at: float = 0.0


def _credentials() -> tuple[str, str] | None:
    username = os.environ.get("BOT_USERNAME", "").strip()
    password = os.environ.get("BOT_PASSWORD", "")
    if not username:
        return None
    return username, password


def _exp_of(token: str) -> float:
    """Reads ``exp`` from the payload (no signature check: the token came
    straight from price-auth; it is only used to know when to renew)."""
    try:
        payload = token.split(".")[1]
        payload += "=" * (-len(payload) % 4)
        return float(json.loads(base64.urlsafe_b64decode(payload))["exp"])
    except (IndexError, KeyError, ValueError):
        return time.time() + 300


def _login(username: str, password: str) -> str | None:
    try:
        resp = httpx.post(
            f"{PRICE_AUTH_URL}/auth/login",
            json={"username": username, "password": password},
            timeout=10,
        )
    except httpx.HTTPError:
        logger.warning("Falha ao chamar price-auth para obter o token do bot.", exc_info=True)
        return None
    if resp.status_code != 200:
        logger.warning("price-auth recusou o login do bot: %s %s", resp.status_code, resp.text)
        return None
    return resp.json().get("access_token")


def auth_headers(force_refresh: bool = False) -> dict[str, str]:
    """``{"Authorization": "Bearer <token>"}``, or ``{}`` when the bot has no
    account configured or price-auth is unreachable."""
    global _token, _expires_at
    creds = _credentials()
    if creds is None:
        return {}
    with _lock:
        if force_refresh or _token is None or time.time() > _expires_at - 60:
            token = _login(*creds)
            if token is None:
                _token, _expires_at = None, 0.0
                return {}
            _token, _expires_at = token, _exp_of(token)
            logger.info("Bot autenticado no price-auth como '%s'.", creds[0])
        return {"Authorization": f"Bearer {_token}"}


def post_with_auth(client: httpx.Client, url: str, **kwargs) -> httpx.Response:
    """POST with the bot's token; on 401 (expired/rotated secret) logs in again
    and retries once."""
    resp = client.post(url, headers=auth_headers(), **kwargs)
    if resp.status_code == 401 and _credentials() is not None:
        resp = client.post(url, headers=auth_headers(force_refresh=True), **kwargs)
    return resp
