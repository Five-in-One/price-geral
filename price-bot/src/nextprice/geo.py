from __future__ import annotations

import json
import logging
import urllib.request
from collections.abc import Callable

logger = logging.getLogger(__name__)

_IPAPI_URL = "https://ipapi.co/json/"
_DEFAULT_LOCATION = "Teresina, PI"


def _fetch_ipapi(url: str) -> dict:
    req = urllib.request.Request(url, headers={"User-Agent": "Mozilla/5.0"})
    with urllib.request.urlopen(req) as response:
        return json.loads(response.read().decode())


def get_user_location(fetch: Callable[[str], dict] = _fetch_ipapi) -> str:
    """Detecta a localização do usuário via IP. `fetch` é injetável para testes."""
    try:
        data = fetch(_IPAPI_URL)
        return f"{data.get('city', 'Teresina')}, {data.get('region', 'PI')}"
    except Exception:
        logger.warning("Falha ao detectar localização, usando padrão.", exc_info=True)
        return _DEFAULT_LOCATION
