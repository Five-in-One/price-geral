from __future__ import annotations

import logging
from datetime import datetime

from playwright.sync_api import Page

from nextprice.stores.base import StoreConfig

logger = logging.getLogger(__name__)


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
