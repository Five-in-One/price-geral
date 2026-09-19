"""Structured data extracted from a supermarket product name.

Scraped names are the only thing the sites give us ("Arroz Branco Camil Tipo 1
5kg", "Cerveja Heineken 350ml Sleek Lt"), and the `unit` field the collector
guesses from them mixes "how it is sold" with "package size". This module turns
a name into fields that can be compared across markets:

- ``brand``           canonical brand, from a lexicon (``data/brands.txt``)
- size                ``pack_count`` x ``unit_amount`` = ``total_amount``, in a
                      base unit (g, ml or m) given by ``unit_kind``
- ``generic_key``     what the product *is*, without brand/size/packaging noise,
                      so two brands of "Feijão Carioca Tipo 1 1kg" share a key
- ``unit_price()``    R$/kg, R$/L, R$/m or R$/un, to compare different sizes

Everything is deterministic (regex + lexicon); no AI involved.
"""

from __future__ import annotations

import re
import unicodedata
from dataclasses import asdict, dataclass, field
from functools import lru_cache
from importlib import resources

# --- text normalisation ---------------------------------------------------------


def fold(text: str) -> str:
    """Lower-case, accent-free text with apostrophes removed and `×` as `x`."""
    decomposed = unicodedata.normalize("NFKD", text)
    folded = "".join(c for c in decomposed if not unicodedata.combining(c)).lower()
    folded = re.sub(r"[’‘`´']", "", folded).replace("×", "x")
    return re.sub(r"\s+", " ", folded).strip()


_TOKEN_RE = re.compile(r"[a-z0-9%]+")


def tokenize(folded: str) -> list[str]:
    return _TOKEN_RE.findall(folded)


# --- units ------------------------------------------------------------------------

# unit word -> (kind, factor to the base unit: g, ml or m)
_UNITS: dict[str, tuple[str, float]] = {
    "kg": ("mass", 1000.0),
    "g": ("mass", 1.0),
    "gr": ("mass", 1.0),
    "grs": ("mass", 1.0),
    "mg": ("mass", 0.001),
    "l": ("volume", 1000.0),
    "lt": ("volume", 1000.0),
    "lts": ("volume", 1000.0),
    "litro": ("volume", 1000.0),
    "litros": ("volume", 1000.0),
    "ml": ("volume", 1.0),
    "m": ("length", 1.0),
    "mt": ("length", 1.0),
    "mts": ("length", 1.0),
}
_BASE_UNIT = {"mass": "g", "volume": "ml", "length": "m"}

_NUM = r"\d+(?:[.,]\d+)?"
_UNIT_ALT = "|".join(sorted(_UNITS, key=len, reverse=True))
# `(?<![\w.,])` keeps us from starting in the middle of a token/number
_MULTIPACK_RE = re.compile(rf"(?<![\w.,])(\d+)\s*x\s*({_NUM})\s*({_UNIT_ALT})\b(?![\w/])")
_MEASURE_RE = re.compile(rf"(?<![\w.,])({_NUM})\s*({_UNIT_ALT})\b(?![\w/])")
_WIDTH_X_LENGTH_RE = re.compile(rf"(?<![\w.,])({_NUM})\s*cm\s*x\s*({_NUM})\s*m\b")
_MODEL_CODE_RE = re.compile(r"\bref\.?\s*\d+")
_APPROX_RE = re.compile(r"\baprox\b\.?")

# "12 un", "8 uni", "2 unis": how many pieces; the measure that follows (if any)
# is then the weight of the whole pack ("Ferrero Rocher 8 Uni 100g").
_UNIT_COUNT_RE = re.compile(r"(?<![\w.,])(\d+)\s*(?:unis|uni|unids|unid|unidades|unds|und|un)\b")
# "12 rolos", "6 saches": pieces whose measure is *per piece* ("30m 12 rolos").
_PIECE_COUNT_RE = re.compile(
    r"(?<![\w.,])(\d+)\s*(?:rolos?|rolinhos?|saches?|capsulas?|pecas?|pares?|tabletes?)\b"
)
# "c/10", "com 32", "cx c/12"
_WITH_COUNT_RE = re.compile(rf"\b(?:c/|com)\s*(\d+)\b(?!\s*(?:{_UNIT_ALT}|%))")

_TYPE_RE = re.compile(r"\b(?:tipo|t)\s*-?\s*(\d)\b")
_WITHOUT_RE = re.compile(r"\bs/\s*")

# --- generic name -----------------------------------------------------------------

# words that carry no product identity
_STOPWORDS = frozenset(
    "a as o os ao aos de da das do dos e em na nas no nos para por com c s p um uma".split()
)
# packaging / listing noise: same product whether or not the word is there
_NOISE = frozenset(
    "lt ln gt lata garrafa long neck sleek pet un uni unis unid unids und unds pct pacote paco "
    "display kit sortido sortidos ref".split()
)
_SYNONYMS = {
    "cero": "zero",
    "alc": "alcool",
    "alcoolico": "alcool",
    "trad": "tradicional",
}


@dataclass(frozen=True)
class UnitPrice:
    value: float
    per: str  # "kg" | "L" | "m" | "un"


@dataclass
class ParsedProduct:
    raw_name: str
    norm_name: str
    brand: str | None = None
    generic_name: str = ""
    generic_key: str = ""
    pack_count: int = 1
    unit_kind: str | None = None  # "mass" | "volume" | "length" | None
    unit_amount: float | None = None  # per piece, in g / ml / m
    total_amount: float | None = None  # whole pack, in g / ml / m
    sold_by_weight: bool = False  # "Banana Prata Kg": the price is per kg
    approximate: bool = False  # "Porção Aprox. 1kg"
    size_label: str = ""
    notes: list[str] = field(default_factory=list)

    @property
    def base_unit(self) -> str | None:
        return _BASE_UNIT.get(self.unit_kind) if self.unit_kind else None

    @property
    def has_size(self) -> bool:
        return self.total_amount is not None or self.pack_count > 1

    def to_dict(self) -> dict:
        return asdict(self)

    def unit_price(self, price: float) -> UnitPrice | None:
        """Price normalised to R$/kg, R$/L, R$/m or R$/un; None if the name
        says nothing about the size."""
        if self.sold_by_weight and self.total_amount is None:
            return UnitPrice(round(price, 2), "kg")
        if self.total_amount and self.unit_kind:
            per, divisor = {
                "mass": ("kg", 1000.0),
                "volume": ("L", 1000.0),
                "length": ("m", 1.0),
            }[self.unit_kind]
            return UnitPrice(round(price / (self.total_amount / divisor), 2), per)
        if "count-only" in self.notes:
            return UnitPrice(round(price / self.pack_count, 2), "un")
        return None


def price_back_fields(parsed: ParsedProduct) -> dict:
    """The parsed data in the shape price-back's POST /api/products accepts
    (optional fields; `size_kind`/`size_amount` only when a size was found)."""
    fields: dict = {
        "brand": parsed.brand or "",
        "generic_key": parsed.generic_key,
        "pack_count": parsed.pack_count,
        "size_label": parsed.size_label,
        "sold_by_weight": parsed.sold_by_weight,
    }
    if parsed.unit_kind and parsed.total_amount:
        fields["size_kind"] = parsed.unit_kind
        fields["size_amount"] = parsed.total_amount
    return fields


# --- brand lexicon -----------------------------------------------------------------


@dataclass(frozen=True)
class _BrandEntry:
    canonical: str
    tokens: tuple[str, ...]
    requires: frozenset[str]


@lru_cache(maxsize=1)
def _load_brands() -> dict[str, list[_BrandEntry]]:
    """First alias token -> candidate entries, longest first."""
    raw = resources.files("nextprice").joinpath("data/brands.txt").read_text(encoding="utf-8")
    index: dict[str, list[_BrandEntry]] = {}
    for line in raw.splitlines():
        line = line.strip()
        if not line or line.startswith("#"):
            continue
        requires: frozenset[str] = frozenset()
        if "@" in line:
            line, req = line.rsplit("@", 1)
            requires = frozenset(tokenize(fold(req)))
        names = [n.strip() for n in line.split("|") if n.strip()]
        canonical = names[0]
        for alias in names:
            tokens = tuple(tokenize(fold(alias)))
            if tokens:
                index.setdefault(tokens[0], []).append(_BrandEntry(canonical, tokens, requires))
    for entries in index.values():
        entries.sort(key=lambda e: len(e.tokens), reverse=True)
    return index


def _find_brands(tokens: list[str]) -> list[tuple[int, int, str]]:
    """Non-overlapping (start, end, canonical) matches, left to right."""
    index = _load_brands()
    token_set = set(tokens)
    found: list[tuple[int, int, str]] = []
    i = 0
    while i < len(tokens):
        for entry in index.get(tokens[i], ()):
            n = len(entry.tokens)
            if tuple(tokens[i : i + n]) == entry.tokens and entry.requires <= token_set:
                found.append((i, i + n, entry.canonical))
                i += n
                break
        else:
            i += 1
    return found


# --- parsing -----------------------------------------------------------------------


def _to_float(text: str) -> float:
    return float(text.replace(",", "."))


def _blank(text: str, span: tuple[int, int]) -> str:
    return text[: span[0]] + " " * (span[1] - span[0]) + text[span[1] :]


def _format_amount(kind: str, amount: float) -> str:
    def num(v: float) -> str:
        return f"{v:.3f}".rstrip("0").rstrip(".").replace(".", ",")

    if kind == "mass":
        return f"{num(amount / 1000)} kg" if amount >= 1000 else f"{num(amount)} g"
    if kind == "volume":
        return f"{num(amount / 1000)} L" if amount >= 1000 else f"{num(amount)} ml"
    return f"{num(amount)} m"


def parse_product_name(name: str) -> ParsedProduct:
    """Extract brand, size and a comparable generic name from a product name."""
    folded = fold(name)
    text = _WITHOUT_RE.sub("sem ", folded)
    text = _TYPE_RE.sub(r"tipo\1", text)
    result = ParsedProduct(raw_name=name, norm_name=folded)

    def take(pattern: re.Pattern[str], current: str) -> tuple[re.Match[str] | None, str]:
        match = pattern.search(current)
        if match:
            current = _blank(current, match.span())
        return match, current

    _, text = take(_MODEL_CODE_RE, text)
    approx, text = take(_APPROX_RE, text)
    result.approximate = approx is not None

    # 1. size: multipack (NxW), width x length, or a single measure
    count: int | None = None
    kind: str | None = None
    per_piece: float | None = None  # amount of one piece, base unit
    whole: float | None = None  # amount of the whole pack, base unit

    multipack, text = take(_MULTIPACK_RE, text)
    dims, text = take(_WIDTH_X_LENGTH_RE, text)
    measure = None
    if multipack:
        count = int(multipack.group(1))
        kind, factor = _UNITS[multipack.group(3)]
        per_piece = _to_float(multipack.group(2)) * factor
    elif dims:
        kind, per_piece = "length", _to_float(dims.group(2))
    else:
        measure, text = take(_MEASURE_RE, text)
        if measure:
            kind, factor = _UNITS[measure.group(2)]
            per_piece = _to_float(measure.group(1)) * factor
            measure_pos = measure.start()

    # 2. piece count ("12 un", "12 rolos", "c/10", "com 32")
    unit_count, text = take(_UNIT_COUNT_RE, text)
    piece_count, text = take(_PIECE_COUNT_RE, text)
    with_count, text = take(_WITH_COUNT_RE, text)
    if count is None:
        for m in (unit_count, piece_count, with_count):
            if m:
                count = int(m.group(1))
                break
        # "8 Uni 100g": the measure after a unit count weighs the *whole* pack
        if unit_count and measure and unit_count.start() < measure_pos and count:
            whole, per_piece = per_piece, (per_piece or 0) / count

    if count is not None and count < 1:
        count = None
    result.pack_count = count or 1

    if kind and per_piece is not None:
        result.unit_kind = kind
        result.unit_amount = round(per_piece, 4)
        total = whole if whole is not None else per_piece * result.pack_count
        result.total_amount = round(total, 4)
        label = _format_amount(kind, result.unit_amount)
        result.size_label = (
            f"{result.pack_count} x {label}" if result.pack_count > 1 else label
        )
    elif count is not None:
        result.size_label = f"{count} un"
        result.notes.append("count-only")

    # 3. sold by weight ("Banana Prata Kg")
    weight_word = re.search(r"\b(?:kg|granel)\b", text)
    if result.total_amount is None and weight_word:
        result.sold_by_weight = True
        result.notes.append("sold-by-weight")
        text = _blank(text, weight_word.span())

    # 4. brand + generic name from what is left
    tokens = tokenize(text)
    brands = _find_brands(tokens)
    if brands:
        best = max(brands, key=lambda b: (b[1] - b[0], -b[0]))
        result.brand = best[2]
    in_brand = {i for start, end, _ in brands for i in range(start, end)}
    generic = [
        _SYNONYMS.get(tok, tok)
        for i, tok in enumerate(tokens)
        if i not in in_brand and tok not in _STOPWORDS and tok not in _NOISE
    ]
    result.generic_name = " ".join(generic)
    result.generic_key = " ".join(sorted(generic))
    return result
