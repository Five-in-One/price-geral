from unittest.mock import MagicMock

import pytest

from nextprice.cli import main as cli_main


def test_build_parser_aceita_extract() -> None:
    args = cli_main.build_parser().parse_args(["extract"])
    assert args.comando == "extract"


def test_build_parser_aceita_track() -> None:
    args = cli_main.build_parser().parse_args(["track"])
    assert args.comando == "track"


def test_build_parser_rejeita_comando_invalido() -> None:
    with pytest.raises(SystemExit):
        cli_main.build_parser().parse_args(["invalido"])


def test_build_parser_exige_um_comando() -> None:
    with pytest.raises(SystemExit):
        cli_main.build_parser().parse_args([])


def test_main_despacha_para_extractor(monkeypatch: pytest.MonkeyPatch) -> None:
    chamado = MagicMock()
    monkeypatch.setattr(cli_main.extractor, "main", chamado)

    cli_main.main(["extract"])

    chamado.assert_called_once_with()


def test_main_despacha_para_tracker(monkeypatch: pytest.MonkeyPatch) -> None:
    chamado = MagicMock()
    monkeypatch.setattr(cli_main.tracker, "main", chamado)

    cli_main.main(["track"])

    chamado.assert_called_once_with()
