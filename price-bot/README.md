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
| Saída | Arquivo JSON datado em `dados_supermercados/` + tabela `scraped_products` no Postgres | Comparativo impresso no terminal + tabela `scraped_products` no Postgres |
| Quando usar | Formar um histórico de preços ao longo do tempo | Consulta pontual: "onde o item X está mais barato agora" |

Os dois fluxos salvam no mesmo lugar: uma tabela de **staging bruto**
(`scraped_products`), não no formato final que o app consome
(`markets`/`products`/`market_products`, servido pela API `price-back`).
Isso é proposital — o bot não sabe decidir sozinho categoria, unidade, ou
se "Tomate" numa loja é o mesmo produto que "Tomate Longa Vida 1kg" em
outra. Ver [Persistência no Postgres](#persistência-no-postgres) abaixo.

A conexão com o Postgres é *best-effort*: se o banco não estiver
acessível, o bot avisa no log e continua funcionando normalmente (JSON
para o extrator, impressão no terminal para o tracker) — só não persiste
no banco naquela execução.

Há um terceiro modo, diferente dos dois acima: a **coleta sob demanda**,
uma API HTTP (`src/nextprice/collect_api.py`) que o Painel Administrativo
(`price-admin-web`) chama quando o admin cola o link de um mercado. Ela
acessa a URL, deixa a IA extrair produto/preço/categoria de forma
genérica (sem termo de busca, sem seletor CSS fixo — o mercado é
desconhecido), salva o bruto em `scraped_products` (`source: 'admin_link'`)
e, diferente dos outros dois fluxos, **também tenta inserir direto** em
`price-back` via `POST /api/products` (tagueado `source: "bot"`) — é o
único dos três fluxos que promove dado da IA para a tabela final sozinho,
sem revisão humana, seguindo o que o documento do projeto descreve para o
Painel Admin.

```bash
python -m nextprice.collect_api   # sobe em :8085
```

Ao enviar cada produto, a coleta também manda o que `src/nextprice/product_parser.py`
extrai do nome (marca, tamanho da embalagem em g/ml/m, quantidade, chave genérica
para comparar marcas diferentes) e o horário da coleta (`collected_at`). Banners de
vitrine ("↪️ Atacado SUPER!") são descartados antes do envio. Marcas novas se ensinam
em `src/nextprice/data/brands.txt` (só marcas, não linhas de produto). Quem usa esses
campos para reconhecer o mesmo produto em mercados diferentes é o catálogo canônico do
`price-back` (veja `../price-back/README.md`): quanto melhor o parser lê marca e tamanho,
mais produtos entram no catálogo sozinhos.

```bash
python scripts/eval_product_parser.py --misses    # mede o parser em tests/data/nomes_reais.txt
python scripts/backfill_parsed_fields.py          # preenche os campos em produtos já cadastrados
```

Requer `price-back` rodando em `:8080` (ou `PRICE_BACK_URL` apontando para
onde ele estiver) e uma `GEMINI_API_KEY` válida — sem uma chave real, a
coleta roda até o passo da IA e retorna `products_extracted: 0` em vez de
travar (testado: com uma chave inválida/placeholder, a rota devolve
`200 OK` com a lista vazia, não um erro).

## Tecnologias utilizadas

| Tecnologia | Função no bot |
| --- | --- |
| Python 3.10+ | Linguagem base do bot |
| Playwright | Automação do navegador (Chromium) para acessar e interagir com os sites das lojas |
| Google Gemini (`google-genai`) | Interpreta os resultados de busca e extrai produto/preço de forma estruturada |
| psycopg | Conecta ao Postgres para persistir os dados coletados |
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
│   ├── db.py
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
├── migrations/
├── tests/
├── dados_supermercados/
└── CLAUDE.md
```

| Caminho | Responsabilidade |
| --- | --- |
| `main.py` | Ponto de entrada único do bot (`extract` / `track`) |
| `extrator.py` | Atalho direto para o fluxo de coleta em lote |
| `querytracker.py` | Atalho direto para o fluxo de busca interativa |
| `src/nextprice/config.py` | Leitura e validação da chave do Gemini |
| `src/nextprice/db.py` | Conexão e migrations do Postgres |
| `src/nextprice/pricing.py` | Conversão de preços em formato brasileiro para número |
| `src/nextprice/storage.py` | Deduplicação/escrita dos arquivos JSON e inserção no Postgres |
| `src/nextprice/geo.py` | Detecção da localização do usuário |
| `src/nextprice/browser.py` | Automação de navegador com Playwright |
| `src/nextprice/product_parser.py` | Extrai marca, tamanho (g/ml/m), quantidade e chave genérica do nome do produto; calcula o preço por kg/L/m/un |
| `src/nextprice/data/brands.txt` | Lista de marcas conhecidas usada pelo parser (uma por linha) |
| `scripts/` | `eval_product_parser.py` (mede o parser) e `backfill_parsed_fields.py` (preenche produtos antigos) |
| `src/nextprice/stores/` | Configuração declarativa de cada loja suportada |
| `src/nextprice/ai/` | Montagem de prompt e cliente do Gemini |
| `src/nextprice/cli/main.py` | Parser de comandos e roteamento (`extract`/`track`) |
| `src/nextprice/cli/extractor.py` | Lógica do fluxo de coleta em lote |
| `src/nextprice/cli/tracker.py` | Lógica do fluxo de busca interativa |
| `migrations/` | Schema SQL da tabela `scraped_products` |
| `tests/` | Testes automatizados |
| `dados_supermercados/` | Saída gerada pelo extrator (não versionada) |
| `CLAUDE.md` | Regras de desenvolvimento do repositório |

## Pré-requisitos

| Requisito | Observação |
| --- | --- |
| Python 3.10 ou superior | Necessário para instalar e rodar o bot |
| Chave de API do Google Gemini | Obtida em https://aistudio.google.com/apikey — necessária apenas para o fluxo `track` |
| Postgres do `price-back` rodando | Opcional — sem ele o bot funciona igual, só não persiste no banco (ver [Persistência no Postgres](#persistência-no-postgres)) |

> [!NOTE]
> O fluxo `extract` funciona sem chave de API — ela só é obrigatória para o
> fluxo `track`, que depende da IA para interpretar os resultados de busca.
> O Postgres é opcional nos dois fluxos: nenhum dos dois exige um banco no
> ar para rodar.

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

| Variável | Descrição | Obrigatória? |
| --- | --- | --- |
| `GEMINI_API_KEY` | Chave de API do Google Gemini, usada pelo `querytracker.py` | Só para `track` |
| `DATABASE_URL` | String de conexão com o Postgres (`price_bot_db`) | Não — tem um padrão pronto para o Postgres do `price-back` local |

> [!IMPORTANT]
> Nunca commite o arquivo `.env` nem cole sua chave em outro arquivo do
> repositório. Ele já está listado no `.gitignore`, mas vale conferir com
> `git status` antes de qualquer commit. Se uma chave for exposta
> acidentalmente, trate-a como comprometida e gere uma nova no Google AI
> Studio.

## Persistência no Postgres

O bot compartilha o mesmo Postgres em Docker usado por `price-back`,
`price-auth` e `price-user` — mas com seu próprio banco (`price_bot_db`),
sem nenhuma tabela em comum com os outros serviços.

```bash
cd ../price-back
docker compose up -d
```

Isso cria `price_bot_db` (uso normal) e `price_bot_db_test` (testes)
automaticamente em uma instalação nova, via
`../price-back/docker/initdb/04-create-bot-db.sql`.

A tabela `scraped_products` (criada por `migrations/0001_init_schema.sql`
na primeira execução) guarda o dado **bruto**, como o bot o vê — não o
formato final que o app consome:

| Coluna | Origem |
| --- | --- |
| `store_key` / `store_name` | `StoreConfig.key` / `.name` da loja |
| `product_name` | Nome do produto, exatamente como veio da página (ou da IA) |
| `price_text` | Preço bruto, ex.: `"R$ 7,79"` |
| `price` | Mesmo preço convertido para número; `NULL` se não der para converter |
| `source` | `'extract'` ou `'track'` |
| `search_term` | Termo buscado (só no fluxo `track`) |
| `collected_at` | Quando a linha foi salva |

Transformar essas linhas brutas no formato de `markets`/`products`/
`market_products` do `price-back` (decidir categoria, unidade, e se dois
nomes de produtos diferentes são o "mesmo" produto) é uma etapa de
curadoria separada — o bot não tem informação suficiente para decidir
isso sozinho.

## Como rodar

O jeito recomendado é pelo ponto de entrada único, `main.py`, que expõe os
dois fluxos como subcomandos:

| Comando | O que faz | Onde ver o resultado |
| --- | --- | --- |
| `python main.py extract` | Varre a vitrine de todas as lojas cadastradas | Arquivos em `dados_supermercados/<loja>_<data>.json` |
| `python main.py track` | Pede um termo de busca e compara preços entre lojas | Comparativo impresso no terminal |
| `python main.py worker` | Consome pedidos de coleta da fila `bot.collect` (RabbitMQ) | Log do terminal; produtos no price-back |
| `python main.py audit` | Assina todos os eventos `collect.*` e só os registra | Log do terminal |
| `python main.py --help` | Lista os subcomandos disponíveis | Ajuda impressa no terminal |

**Worker de coleta (Entrega 3 — comunicação assíncrona):** o Painel Admin
pede a coleta ao `price-back`, que responde 202 e publica `collect.requested`
na exchange `price.events`. O worker (`src/nextprice/worker.py`) recebe a
mensagem, roda a mesma coleta do `POST /collect` e publica
`collect.finished` com o resultado. Rode quantos workers quiser (um por
terminal) para dividir a carga. Precisa do RabbitMQ do `docker compose` do
price-back e de `RABBITMQ_URL` no `.env`. Mensagens inválidas vão para a
fila `bot.collect.dlq`.

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

> [!NOTE]
> Os testes em `tests/test_db_storage.py` usam o Postgres real
> (`price_bot_db_test`) e pulam automaticamente (`SKIPPED`) se o
> `docker compose up -d` do `price-back` não estiver rodando.

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
