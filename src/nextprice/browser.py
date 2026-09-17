from __future__ import annotations

import logging
import re
from datetime import datetime

from playwright.sync_api import Page

from nextprice.stores.base import StoreConfig
from nextprice.text import normalize

logger = logging.getLogger(__name__)

# Banners promocionais nesses sites ("Atacado SUPER!", "Da Nossa Família
# para Sua!", emojis à parte) reaproveitam a mesma classe CSS dos cards de
# produto de verdade — não dá pra distinguir só pelo seletor. Na prática,
# produtos de mercearia raspados aqui nunca têm emoji no nome; banners
# promocionais quase sempre têm. Cobre os blocos Unicode de emoji mais
# comuns (pictográficos, símbolos diversos/dingbats, bandeiras).
_EMOJI_PATTERN = re.compile(
    "[" "\U0001f300-\U0001faff" "\U00002600-\U000027bf" "\U0001f1e6-\U0001f1ff" "]"
)

# Nem todo banner tem emoji ("Ofertas do dia", "Mais Vendidos", "Peso
# Variável", "Exclusivo Delivery" passaram batido). Essas são frases de
# vitrine/UI que nunca aparecem como nome de produto de mercearia — lista
# comparada sem acento (ver _normalize) para cobrir variações de grafia.
_BANNER_KEYWORDS = (
    "oferta",
    "promocao",
    "mais vendido",
    "destaque",
    "exclusivo",
    "delivery",
    "peso vari",  # "peso variável" / "peso variavel"
    "novidade",
    "desconto",
    "frete gratis",
    "cupom",
    "confira",
    "aproveite",
    "leve mais",
    "compre e leve",
)


def _looks_like_banner(nome: str) -> bool:
    """Heurística para filtrar banners/rótulos de vitrine capturados junto
    com produtos de verdade pelo mesmo seletor CSS — por emoji
    (_EMOJI_PATTERN) ou por frase típica de banner (_BANNER_KEYWORDS)."""
    if _EMOJI_PATTERN.search(nome):
        return True
    return any(palavra in normalize(nome) for palavra in _BANNER_KEYWORDS)


def scroll_to_load(
    page: Page, times: int = 5, wheel_amount: int = 1000, wait_ms: int = 2000
) -> None:
    """Rola a página para forçar o carregamento lazy da vitrine de produtos."""
    for _ in range(times):
        page.mouse.wheel(0, wheel_amount)
        page.wait_for_timeout(wait_ms)


def scrape_listing(page: Page, store: StoreConfig) -> list[dict]:
    """Extrai produtos (nome/preço) da vitrine atual da página, usando os seletores da loja.

    Função genérica que substitui as antigas raspar_carvalho/raspar_ferreira:
    adicionar uma loja nova não exige duplicar esta lógica, só um StoreConfig novo.
    """
    produtos: list[dict] = []
    cartoes = page.locator(store.card_selector).all()

    for cartao in cartoes:
        try:
            preco_el = cartao.locator(store.price_selector).first
            if preco_el.count() == 0:
                continue
            preco = preco_el.inner_text().strip()
            if store.require_price_prefix and store.require_price_prefix not in preco:
                continue

            nome_el = cartao.locator(store.name_selector).first
            if nome_el.count() == 0:
                continue
            nome = nome_el.inner_text().strip()

            if len(nome) <= 3 or nome == preco:
                continue
            if _looks_like_banner(nome):
                continue
            nome_lower = nome.lower()
            if any(termo in nome_lower for termo in store.exclude_name_substrings):
                continue

            produtos.append(
                {
                    "nome": nome,
                    "preco": preco,
                    "data_coleta": datetime.now().strftime("%Y-%m-%d %H:%M:%S"),
                }
            )
        except Exception:
            logger.debug(
                "Falha ao processar um cartão de %s, ignorando.", store.name, exc_info=True
            )
            continue

    return produtos


_GENERIC_CARD_SELECTOR = (
    "div.ng-star-inserted:has-text('R$'), article, .product-item, "
    "div[class*='product'], li[class*='product']"
)


def collect_listing_text_blocks(page: Page, url: str, max_block_len: int = 300) -> list[str]:
    """Visits an arbitrary market URL and collects text blocks that look
    like product cards (contain "R$"), for the admin panel's on-demand
    collection — no StoreConfig/CSS selectors registered for this site, so
    this uses the same broad heuristic as search_and_collect_cards and lets
    the AI sort out what's actually a product.
    """
    page.goto(url, timeout=60000)
    page.wait_for_timeout(4000)
    scroll_to_load(page, times=3, wheel_amount=1000, wait_ms=1500)

    blocos: list[str] = []
    elementos = page.locator(_GENERIC_CARD_SELECTOR).all()
    for el in elementos:
        try:
            texto = el.inner_text().strip()
            if (
                "R$" in texto
                and 0 < len(texto) < max_block_len
                and texto not in blocos
                and not _looks_like_banner(texto)
            ):
                blocos.append(texto)
        except Exception:
            logger.debug("Falha ao ler um bloco de %s, ignorando.", url, exc_info=True)
            continue

    return blocos


def search_and_collect_cards(page: Page, store: StoreConfig, termo: str) -> list[str]:
    """Abre a busca da loja pelo termo e coleta blocos de texto candidatos a produto.

    Usado pelo fluxo do querytracker, que depois envia os blocos para a IA
    interpretar (a estrutura HTML de resultado de busca varia mais entre lojas
    do que a vitrine, então aqui delegamos a extração estruturada à IA).
    """
    page.goto(store.base_url, timeout=60000)
    page.wait_for_timeout(4000)

    input_pesquisa = page.locator(store.search_input_selector)
    if input_pesquisa.count() > 0:
        logger.info("[%s] Campo de pesquisa encontrado! Digitando '%s'...", store.name, termo)
        input_pesquisa.click()
        input_pesquisa.fill(termo)
        page.wait_for_timeout(1000)
        page.keyboard.press("Enter")
        page.wait_for_timeout(7000)
    else:
        logger.warning("[%s] Input de busca não encontrado, usando URL direta...", store.name)
        page.goto(f"{store.base_url}?s={termo}", timeout=60000)
        page.wait_for_timeout(6000)

    scroll_to_load(page, times=3, wheel_amount=1000, wait_ms=1500)

    cards_textos: list[str] = []
    elementos = page.locator(store.search_card_selector).all()
    termo_lower = termo.lower()
    for el in elementos:
        try:
            texto_card = el.inner_text().strip()
            if termo_lower in texto_card.lower() and "R$" in texto_card and len(texto_card) < 300:
                if texto_card not in cards_textos:
                    cards_textos.append(texto_card)
        except Exception:
            logger.debug(
                "Falha ao ler um card de busca de %s, ignorando.", store.name, exc_info=True
            )
            continue

    return cards_textos
