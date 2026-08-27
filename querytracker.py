"""Wrapper fino para manter `python querytracker.py` funcionando sem instalar o pacote.

A lógica real vive em src/nextprice/cli/tracker.py.
"""

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent / "src"))

from nextprice.cli.tracker import main  # noqa: E402

if __name__ == "__main__":
    main()
