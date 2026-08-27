from __future__ import annotations

import argparse

from nextprice.cli import extractor, tracker


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="nextprice",
        description="Bot de coleta e comparação de preços de supermercados.",
    )
    subparsers = parser.add_subparsers(dest="comando", required=True)

    subparsers.add_parser(
        "extract",
        help="Varre a vitrine de todas as lojas cadastradas e salva um snapshot em JSON.",
    )
    subparsers.add_parser(
        "track",
        help="Busca interativa por um termo e compara preços entre lojas usando IA.",
    )

    return parser


def main(argv: list[str] | None = None) -> None:
    parser = build_parser()
    args = parser.parse_args(argv)

    if args.comando == "extract":
        extractor.main()
    elif args.comando == "track":
        tracker.main()


if __name__ == "__main__":
    main()
