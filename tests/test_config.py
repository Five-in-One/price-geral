from pathlib import Path

import pytest

from nextprice.config import Settings


def test_settings_load_le_variavel_de_ambiente(
    monkeypatch: pytest.MonkeyPatch, tmp_path: Path
) -> None:
    monkeypatch.chdir(tmp_path)
    monkeypatch.setenv("GEMINI_API_KEY", "chave-de-teste")

    settings = Settings.load()

    assert settings.gemini_api_key == "chave-de-teste"


def test_settings_load_falha_sem_chave(monkeypatch: pytest.MonkeyPatch, tmp_path: Path) -> None:
    monkeypatch.chdir(tmp_path)
    monkeypatch.delenv("GEMINI_API_KEY", raising=False)

    with pytest.raises(RuntimeError, match="GEMINI_API_KEY"):
        Settings.load()
