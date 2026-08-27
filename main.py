"""Ponto de entrada único do bot: `python main.py extract` ou `python main.py track`.

Wrapper fino para manter o comando funcionando sem instalar o pacote.
A lógica real vive em src/nextprice/cli/main.py.
"""

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent / "src"))

from nextprice.cli.main import main  # noqa: E402

if __name__ == "__main__":
    main()
