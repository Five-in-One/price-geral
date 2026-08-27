# NextPrice Bot

Bot para coletar e comparar preços de produtos em supermercados locais,
combinando web scraping com interpretação por IA.

## Sobre o bot

O NextPrice Bot automatiza duas tarefas que, feitas manualmente, tomariam
tempo: acompanhar o preço de produtos ao longo do tempo e comparar preços
entre lojas na hora de decidir onde comprar. O bot acessa diretamente os
sites dos supermercados via automação de navegador, sem depender de APIs
oficiais que essas lojas não oferecem.

Ele é composto por dois fluxos independentes:

| Aspecto | Extrator (`extrator.py`) | Query Tracker (`querytracker.py`) |
| --- | --- | --- |
| Execução | Em lote, pensado para rodar periodicamente | Interativa, pede um termo de busca ao usuário |
| Fonte dos dados | Vitrine/página inicial de cada loja | Página de resultado de busca de cada loja |
| Extração | Seletores CSS fixos por loja | IA (Gemini) interpreta o texto coletado |
| Navegador | Headless (sem interface visível) | Visível, para acompanhar a busca em tempo real |
| Saída | Arquivo JSON datado em `dados_supermercados/` | Comparativo de preços impresso no terminal |
| Quando usar | Formar um histórico de preços ao longo do tempo | Consulta pontual: "onde o item X está mais barato agora" |

## Tecnologias utilizadas

| Tecnologia | Função no bot |
| --- | --- |
| Python 3.10+ | Linguagem base do bot |
| Playwright | Automação do navegador (Chromium) para acessar e interagir com os sites das lojas |
| Google Gemini (`google-genai`) | Interpreta os resultados de busca e extrai produto/preço de forma estruturada |
| python-dotenv | Carrega variáveis de ambiente a partir de um arquivo `.env` |
| pytest | Testes automatizados da lógica de negócio |
| ruff | Lint e formatação de código |
| mypy | Checagem estática de tipos |
| hatchling | Build backend do pacote Python |

## Estrutura do bot

```
NextPrice/
├── main.py
├── extrator.py
├── querytracker.py
├── src/nextprice/
│   ├── config.py
│   ├── pricing.py
│   ├── storage.py
│   ├── geo.py
│   ├── browser.py
│   ├── stores/
│   ├── ai/
│   └── cli/
│       ├── main.py
│       ├── extractor.py
│       └── tracker.py
├── tests/
├── dados_supermercados/
└── CLAUDE.md
```

| Caminho | Responsabilidade |
| --- | --- |
| `main.py` | Ponto de entrada único do bot (`extract` / `track`) |
| `extrator.py` | Atalho direto para o fluxo de coleta em lote |
| `querytracker.py` | Atalho direto para o fluxo de busca interativa |
| `src/nextprice/config.py` | Leitura e validação de variáveis de ambiente |
| `src/nextprice/pricing.py` | Conversão de preços em formato brasileiro para número |
| `src/nextprice/storage.py` | Deduplicação e escrita dos arquivos JSON |
| `src/nextprice/geo.py` | Detecção da localização do usuário |
| `src/nextprice/browser.py` | Automação de navegador com Playwright |
| `src/nextprice/stores/` | Configuração declarativa de cada loja suportada |
| `src/nextprice/ai/` | Montagem de prompt e cliente do Gemini |
| `src/nextprice/cli/main.py` | Parser de comandos e roteamento (`extract`/`track`) |
| `src/nextprice/cli/extractor.py` | Lógica do fluxo de coleta em lote |
| `src/nextprice/cli/tracker.py` | Lógica do fluxo de busca interativa |
| `tests/` | Testes automatizados |
| `dados_supermercados/` | Saída gerada pelo extrator (não versionada) |
| `CLAUDE.md` | Regras de desenvolvimento do repositório |

## Pré-requisitos

| Requisito | Observação |
| --- | --- |
| Python 3.10 ou superior | Necessário para instalar e rodar o bot |
| Chave de API do Google Gemini | Obtida em https://aistudio.google.com/apikey — necessária apenas para o fluxo `track` |

> [!NOTE]
> O fluxo `extract` funciona sem chave de API — ela só é obrigatória para o
> fluxo `track`, que depende da IA para interpretar os resultados de busca.

## Instalação

Clone o repositório e, dentro da pasta do bot, crie um ambiente virtual:

```bash
python -m venv .venv
```

Ative o ambiente virtual de acordo com o seu sistema operacional:

| Sistema operacional | Comando |
| --- | --- |
| Windows (PowerShell) | `.venv\Scripts\Activate.ps1` |
| Windows (cmd) | `.venv\Scripts\activate.bat` |
| Linux/macOS | `source .venv/bin/activate` |

Instale o bot e suas dependências:

```bash
pip install -e ".[dev]"
```

Instale o navegador usado pelo Playwright:

```bash
playwright install chromium
```

## Configuração

Copie o arquivo de exemplo e preencha com sua chave da API:

```bash
cp .env.example .env
```

| Variável | Descrição |
| --- | --- |
| `GEMINI_API_KEY` | Chave de API do Google Gemini, usada pelo `querytracker.py` |

> [!IMPORTANT]
> Nunca commite o arquivo `.env` nem cole sua chave em outro arquivo do
> repositório. Ele já está listado no `.gitignore`, mas vale conferir com
> `git status` antes de qualquer commit. Se uma chave for exposta
> acidentalmente, trate-a como comprometida e gere uma nova no Google AI
> Studio.

## Como rodar

O jeito recomendado é pelo ponto de entrada único, `main.py`, que expõe os
dois fluxos como subcomandos:

| Comando | O que faz | Onde ver o resultado |
| --- | --- | --- |
| `python main.py extract` | Varre a vitrine de todas as lojas cadastradas | Arquivos em `dados_supermercados/<loja>_<data>.json` |
| `python main.py track` | Pede um termo de busca e compara preços entre lojas | Comparativo impresso no terminal |
| `python main.py --help` | Lista os subcomandos disponíveis | Ajuda impressa no terminal |

Os scripts `extrator.py` e `querytracker.py` continuam funcionando como
atalhos diretos para cada fluxo individual, se preferir:

| Comando | Equivalente a |
| --- | --- |
| `python extrator.py` | `python main.py extract` |
| `python querytracker.py` | `python main.py track` |

> [!WARNING]
> O fluxo `track` abre um navegador visível e faz chamadas reais à API do
> Gemini a cada busca — cada execução consome cota da sua chave de API.

## Testes e qualidade de código

| Comando | Finalidade |
| --- | --- |
| `pytest` | Roda a suíte de testes automatizados |
| `ruff check .` | Verifica lint |
| `ruff format .` | Formata o código automaticamente |
| `mypy src` | Verifica tipos estaticamente |

## Adicionando uma nova loja

O bot foi desenhado para que uma nova loja não exija duplicar código de
scraping. Basta:

1. Criar um novo arquivo em `src/nextprice/stores/` com um `StoreConfig`
   descrevendo URL e seletores da loja.
2. Registrar esse `StoreConfig` na lista `STORES` em
   `src/nextprice/stores/registry.py`.

> [!TIP]
> Os campos disponíveis em `StoreConfig` (seletores de card, preço, nome,
> filtros de exclusão etc.) estão documentados em
> `src/nextprice/stores/base.py`. Não é necessário escrever uma nova função
> de scraping — `browser.scrape_listing` e `browser.search_and_collect_cards`
> já são genéricas e reaproveitadas por todas as lojas.
