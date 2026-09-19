"""Mede o parser de nomes de produto contra nomes reais coletados.

Uso (na pasta price-bot):
    .venv/Scripts/python.exe -X utf8 scripts/eval_product_parser.py [arquivo.txt] [--misses]

O arquivo tem um nome por linha (padrão: tests/data/nomes_reais.txt). Mostra
quantos nomes saíram com tamanho e com marca e, com --misses, lista os que
ficaram sem — é a lista de trabalho para ensinar marcas novas em
src/nextprice/data/brands.txt ou ajustar o parser.
"""

from __future__ import annotations

import sys
from collections import Counter
from pathlib import Path

from nextprice.product_parser import parse_product_name

DEFAULT_CORPUS = Path(__file__).resolve().parents[1] / "tests" / "data" / "nomes_reais.txt"


def main() -> None:
    args = [a for a in sys.argv[1:] if not a.startswith("--")]
    show_misses = "--misses" in sys.argv
    corpus = Path(args[0]) if args else DEFAULT_CORPUS
    names = [n.strip() for n in corpus.read_text(encoding="utf-8").splitlines() if n.strip()]

    parsed = [parse_product_name(n) for n in names]
    total = len(parsed)

    with_size = [p for p in parsed if p.total_amount is not None]
    count_only = [p for p in parsed if p.total_amount is None and p.pack_count > 1]
    by_weight = [p for p in parsed if p.sold_by_weight]
    no_size = [p for p in parsed if not p.has_size and not p.sold_by_weight and p.pack_count == 1]
    with_brand = [p for p in parsed if p.brand]
    no_brand = [p for p in parsed if not p.brand]

    def pct(n: int) -> str:
        return f"{n:4d}  ({100 * n / total:5.1f}%)"

    print(f"Nomes avaliados: {total}\n")
    print(f"tamanho medido (g/ml/m)    {pct(len(with_size))}")
    print(f"só contagem (12 un, c/10)  {pct(len(count_only))}")
    print(f"vendido por kg             {pct(len(by_weight))}")
    print(f"SEM tamanho nenhum         {pct(len(no_size))}")
    print(f"marca identificada         {pct(len(with_brand))}")
    print(f"SEM marca                  {pct(len(no_brand))}")

    keys = Counter(p.generic_key for p in parsed if p.generic_key)
    shared = sum(1 for c in keys.values() if c > 1)
    print(f"\nchaves genéricas distintas: {len(keys)} ({shared} compartilhadas por 2+ nomes)")

    if show_misses:
        print("\n--- SEM tamanho nenhum ---")
        for p in no_size:
            print(f"  {p.raw_name}")
        print("\n--- SEM marca ---")
        for p in no_brand:
            print(f"  {p.raw_name}")


if __name__ == "__main__":
    main()
