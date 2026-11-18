# price-back

API de Produtos e Preços do Price (Go + chi + pgx), com PostgreSQL rodando
em Docker. Cuida só de mercados, produtos e comparação de preços — login
fica em `price-auth` e histórico/economia do usuário em `price-user`
(ambos serviços separados que compartilham este mesmo Postgres).

## Entrega 3 — Comunicação assíncrona (DSD 2026.2)

O fluxo **"Coletar mercado"** do Painel Admin deixou de ser uma chamada REST
síncrona ao bot (que levava minutos e falhava se o bot estivesse fora do ar)
e passou a usar **RabbitMQ**. Os detalhes estão em
[Coleta assíncrona (RabbitMQ)](#coleta-assíncrona-rabbitmq).

| Papel | Onde | Repositório |
| --- | --- | --- |
| Broker | RabbitMQ no `docker-compose.yml` | este |
| **Produtor** | `POST /api/admin/collect`: Controller `internal/httpserver/collect.go` → Service `internal/service/collect.go` (salva mercado + job no banco e publica `collect.requested`) → **202 Accepted** | este |
| **Consumidor** | Worker na fila `bot.collect`: `src/nextprice/worker.py` desserializa o DTO e chama o service `run_collection` | [price-bot](https://github.com/Five-in-One/price-bot) |
| Retorno do resultado | `collect.finished` → consumer `internal/messaging/consumer.go` → Service fecha o job | este |
| Painel | "Coletar"/"Recarregar" recebem 202 e consultam o job a cada 2s | [price-front-admin](https://github.com/Five-in-One/price-front-admin) |

**Padrões de distribuição:**
- **Point-to-Point:** vários workers na mesma fila `bot.collect` (`prefetch=1`). Cada mensagem vai para um só, com balanceamento de carga.
- **Publish/Subscribe:** a exchange `price.events` é do tipo *topic*. A fila `audit.collect` (`python main.py audit`) assina `collect.*` e recebe uma cópia de todos os eventos, sem afetar os workers.

O produtor só conhece a exchange `price.events` e a routing key. Ele não sabe
quais filas ou serviços consomem o evento.

### Como rodar a demonstração

```bash
# 1. Broker + banco (neste repo)
docker compose up -d
# 2. API (neste repo) — porta 8080
go run ./cmd/api
# 3. Dois workers + o assinante de auditoria (no repo price-bot, um por terminal)
python main.py worker
python main.py worker
python main.py audit
# 4. Painel (repo price-front-admin) — http://localhost:8090, com price-auth na 8081
python -m http.server 8090
```

Para ver acontecendo:
- **202 na hora:** clique em "Coletar" no painel. O pedido é aceito na hora e a coleta roda em segundo plano.
- **Balanceamento:** dispare vários pedidos ou um "Recarregar". Os logs mostram os jobs divididos entre os workers (`[worker pid=...]`).
- **Resiliência:** derrube os workers e peça uma coleta. Ainda vem 202, e a mensagem fica parada em `bot.collect` (veja em http://localhost:15672, guest/guest). Suba um worker e ela é processada.

Teste sem o painel:

```bash
curl -X POST localhost:8080/api/admin/collect -H "Content-Type: application/json" \
  -d '{"market_name":"Mercado Teste","url":"https://exemplo.com.br/ofertas"}'
curl localhost:8080/api/collect/jobs/<job_id>
```

## Subir o banco

```bash
docker compose up -d
```

Isso sobe o Postgres compartilhado pelos três serviços (`price-back`,
`price-auth`, `price-user`) e cria os bancos de cada um: `price_db` /
`price_db_test` (este serviço), `price_auth_db` / `price_auth_db_test` e
`price_user_db` / `price_user_db_test`.

O mesmo compose sobe o **RabbitMQ** (`5672`; painel em
http://localhost:15672, usuário/senha `guest`), usado na coleta assíncrona
(veja abaixo).

## Coleta assíncrona (RabbitMQ)

A coleta pedida pelo Painel Admin não roda mais dentro da requisição HTTP:

```
POST /api/admin/collect ──▶ price-back: salva mercado + job "queued"
                            publica collect.requested ──▶ exchange price.events (topic)
                            responde 202 + job                  │
                                                                ├─▶ fila bot.collect  ─▶ workers do price-bot (point-to-point)
                                                                └─▶ fila audit.collect ─▶ price-bot audit (pub/sub, só log)
price-bot worker: coleta, publica collect.finished ──▶ fila back.collect-status ─▶ price-back fecha o job
GET /api/collect/jobs/{id} ◀── o painel consulta a cada 2s
```

- O `price-back` só conhece a exchange e a routing key, nunca quem consome.
- Vários workers (`python main.py worker` no price-bot) dividem a fila
  `bot.collect`: cada mensagem vai para um só (`prefetch=1`).
- Com o bot desligado, o pedido continua recebendo 202 e a mensagem espera
  na fila até um worker subir.
- Código: `internal/messaging` (broker, eventos, consumer), `internal/service/collect.go` (regras) e
  `internal/httpserver/collect.go` (rotas).

| Rota | Resposta |
| --- | --- |
| `POST /api/admin/collect` `{url, market_name, cep?, street?, ...}` | **202** `{market, job}` · 409 `collect_in_progress` (traz o job existente) · 503 `broker_unavailable` |
| `POST /api/markets/{id}/collect` ("Recarregar") | **202** `{market_id, jobs}`, um job por link |
| `GET /api/collect/jobs/{id}` | `{id, status: queued\|succeeded\|failed, products_inserted, error, ...}` |

## Rodar a API

```bash
go run ./cmd/api
```

As migrations em `migrations/*.sql` são aplicadas automaticamente na
inicialização. A API sobe em `:8080` por padrão (`PORT` no ambiente para
mudar). O banco de produção não recebe mais dados de demonstração; o
dataset usado pelos testes fica em `internal/testutil`.

A migration `0009` cria a extensão `pg_trgm` (`CREATE EXTENSION`), então o
usuário do banco precisa ter permissão para isso — no Postgres do
`docker-compose.yml` o usuário `postgres` já tem.

## Rodar os testes

Com o `docker compose up -d` já rodando:

```bash
go test ./...
```

Os testes de integração usam `price_db_test`, migram e resetam as tabelas
para o estado seedado antes de cada teste.

## Rotas

| Método | Rota | Descrição |
|---|---|---|
| GET | `/health` | healthcheck |
| GET | `/api/markets?lat=&lng=&radius_km=` | mercados dentro do raio (padrão 5km), ordenados por distância |
| GET | `/api/markets?all=true&search=` | todos os mercados (ignora raio), com busca por nome — usado pelo Painel Admin |
| POST | `/api/markets` | cria um mercado (ou reaproveita um já existente com o mesmo nome) — `{name, latitude?, longitude?, initials?, primary_color?, background_color?, source?, link?, cep?, street?, number?, neighborhood?, city?, state?, complement?}`. Campos de endereço se mesclam num resubmit pelo mesmo nome: um valor não-vazio substitui o antigo, um vazio nunca apaga o que já estava salvo. `link` (uma URL por vez) se acumula em `links` — o mesmo mercado pode ter mais de um site (a lista retornada nunca duplica a mesma URL); a identidade do mercado continua sendo o nome, nunca a URL |
| POST | `/api/markets/{id}/links` | adiciona um link ao mercado — `{url}`. Idempotente (resubmeter a mesma URL não duplica) |
| DELETE | `/api/markets/{id}/links?url=` | remove um link do mercado (comparação sem diferenciar maiúsculas/minúsculas) |
| PATCH | `/api/markets/{id}` | edita nome, iniciais, cor e **endereço** (`cep`, `street`, `number`, `neighborhood`, `city`, `state`, `complement`). Se o endereço mudou, ele é geocodificado de novo (veja [Localização e distância](#localização-e-distância-dos-mercados)) |
| DELETE | `/api/markets/{id}` | remove o mercado e os produtos que têm ele como mercado padrão |
| GET | `/api/products?category=&search=&sort=price_asc&source=` | busca/filtra produtos (filtro `source`: `seed`\|`company`\|`bot`). Cada produto traz também os [campos extraídos do nome](#dados-extraídos-do-nome-do-produto), `unit_price` e `catalog_id` |
| POST | `/api/products` | cria/atualiza um produto e seu preço em um mercado — `{name, unit, category, price, market_id\|market_name, source?}` mais os campos opcionais do parser. Já liga o produto ao [catálogo](#catálogo-canônico) |
| PATCH | `/api/products/{id}` | edita nome/unidade/categoria/preço e os campos do parser (usado no backfill de produtos antigos) |
| DELETE | `/api/products/{id}` | remove o produto e seus preços |
| POST | `/api/comparisons` | compara o custo de uma lista `{items:[{product_id,quantity}]}` entre todos os mercados |
| POST | `/api/comparisons/list` | compara uma lista de itens do catálogo: total por mercado, itens que faltam, mais barato por item e melhor combinação — veja [Comparação de listas](#comparação-de-listas) |
| GET | `/api/catalog/items?search=&min_markets=&limit=&offset=` | itens do [catálogo canônico](#catálogo-canônico) com as ofertas de cada mercado, mais baratas primeiro |
| GET | `/api/catalog/items/{id}` | um item do catálogo |
| POST | `/api/catalog/match?all=` | liga ao catálogo os produtos que ainda não estão nele (idempotente); `all=true` reavalia todos |
| GET | `/api/catalog/suggestions?status=` | fila de revisão (padrão `pending`) |
| POST | `/api/catalog/suggestions/{id}/accept` | "são o mesmo produto": junta e trava a decisão |
| POST | `/api/catalog/suggestions/{id}/reject` | "não são o mesmo": nunca mais é sugerido |
| GET | `/api/geocode/reverse?lat=&lng=` | endereço de uma posição (`street`, `neighborhood`, `city`, `state`, `cep` e `label` numa linha só), via OpenStreetMap Nominatim. O app usa depois de ler o GPS para mostrar onde a pessoa está. `400` sem coordenadas válidas, `404` se não há endereço ali, `502` se o Nominatim falhar |

Histórico de economia (`/api/savings-history`, `/api/economy/summary`)
mudou para o serviço `price-user` — veja `../price-user/README.md`.

As rotas `POST /api/markets` e `POST /api/products` alimentam duas telas
Web (fora deste repo): `price-admin-web` (Painel Administrativo) e
`price-empresa-web` (Página da Empresa). Cada mercado/produto criado por
elas carrega `source: "company"` ou `"bot"` para diferenciar da massa de
dados seedada (`source: "seed"`).

## Dados extraídos do nome do produto

O bot (`price-bot`) analisa o nome de cada produto coletado
("Café Melitta Tradicional Vácuo 500g") e envia, junto com o preço, o que
conseguiu extrair. Todos os campos são **opcionais** — produtos cadastrados
pela página da empresa simplesmente não os têm, e um reenvio sem eles nunca
apaga o que o bot já gravou.

| Campo | Exemplo | O que é |
|---|---|---|
| `brand` | `Melitta` | marca, a partir de uma lista de marcas conhecidas |
| `generic_key` | `cafe tradicional vacuo` | o que o produto *é*, sem marca, tamanho nem embalagem — palavras sem acento, ordenadas |
| `size_kind` + `size_amount` | `mass` + `500` | tamanho da embalagem inteira numa unidade-base: `mass` = g, `volume` = ml, `length` = m. Vêm juntos ou não vêm |
| `pack_count` | `12` | peças na embalagem (12 rolos, 30 barras); padrão 1 |
| `size_label` | `12 x 30 m` | texto pronto para mostrar |
| `sold_by_weight` | `true` | "Queijo … KG": o preço listado é por kg |
| `collected_at` | `2026-09-19T10:00:00Z` | quando o preço foi observado (padrão: agora) |

`unit_price` / `unit_price_unit` são **calculados na leitura** (nunca
gravados) a partir do preço e do tamanho: R$/kg, R$/L, R$/m ou R$/un. É o
que permite comparar embalagens de tamanhos diferentes — o café de 250 g a
R$ 12,90 sai a R$ 51,60/kg e o de 500 g a R$ 28,90 sai a R$ 57,80/kg.

Validações do `POST`/`PATCH`: `size_kind` e `size_amount` andam juntos,
`size_kind` ∈ {`mass`, `volume`, `length`}, `pack_count` ≥ 1 e
`collected_at` não pode estar no futuro. Um `PATCH` com novo `price`
atualiza `collected_at`; um `PATCH` que só preenche os campos do parser
(backfill) **não** mexe nele — preço antigo não pode parecer recém-coletado.

## Catálogo canônico

### A ideia

Cada mercado cadastra seus produtos por conta própria, então o mesmo café
aparece como duas linhas sem relação entre si — e por isso uma lista de
compras não tinha como ser comparada entre mercados. O catálogo canônico dá
**uma ficha única para cada produto do mundo real**, e cada linha de mercado
(a *oferta*) aponta para ela:

```
catalog_items                           products (oferta de um mercado)
"Bebida Láctea Nescau 180ml"  ◄──────   Atacadão  · R$ 2,29   (catalog_id)
                              ◄──────   Carvalho  · R$ 2,79   (catalog_id)
```

Nenhum produto existente é alterado ou removido: só ganha o vínculo
`products.catalog_id`.

### O que faz dois cadastros serem "o mesmo produto"

A identidade de um item é **marca + chave genérica + tamanho da embalagem
(tipo, quantidade, nº de peças) + vendido por kg**, tudo vindo do parser do
bot. O tamanho é regra obrigatória.

| Caso | Resultado |
|---|---|
| "Café Melitta Vácuo 500g" e "Cafe Melitta Vacuo 500g" | **mesmo item** (acento e caixa não contam) |
| "Maionese Heinz 390g" e "Maionese Heinz 390gr" | **mesmo item** |
| Café Melitta 250 g × 500 g | itens diferentes — tamanho diferente (só a similaridade de nome juntaria os dois) |
| Arroz Camil 5 kg × Arroz Sofia 5 kg | itens diferentes — marca diferente. São *equivalentes*, não o *mesmo produto* (outra funcionalidade) |
| Cerveja Heineken × Cerveja Heineken Zero | itens diferentes — variantes |

### Como a ligação acontece

1. **Automática, ao cadastrar.** `POST /api/products` (e `PATCH` com campos do
   parser) chama a ligação logo após salvar o preço. Se der erro, o preço é
   salvo do mesmo jeito e o erro só vai para o log — `POST /api/catalog/match`
   recupera depois.
2. **Backfill.** `POST /api/catalog/match` liga tudo que ainda não está no
   catálogo. É idempotente; `?all=true` reavalia todos os produtos que uma
   pessoa não travou.
3. Se a identidade bate com um item existente, a oferta entra nele; senão um
   item novo é criado. Se a identidade de um produto mudar (o parser melhorou),
   ele muda de item e o item que ficou vazio é apagado.

Produtos sem `generic_key` (por exemplo, os da página da empresa) não são
ligados — aparecem como `skipped` na resposta do `match`.

### Fila de revisão

Ligar dois cadastros errados é o pior erro possível: o comparador trataria
coisas diferentes como se fossem iguais. Por isso o sistema **nunca junta por
palpite**. Quando cria um item novo, ele procura itens de **mesmo tamanho** e
enfileira um pedido de revisão só em dois casos:

| `reason` | Quando |
|---|---|
| `brand-missing` | mesma chave genérica, mas a marca falta num dos nomes ("Feijão Carioca Tipo 1 1kg" × "Feijão Carioca Sofia Tipo 1 1kg") |
| `similar-name` | marcas iguais (ou uma ausente) e nomes com similaridade `pg_trgm` ≥ 0,85 ("Corona Cero 330ml 0% Alc" × "Corona Zero Álcool 330ml") |

O limiar é alto de propósito. Medido nos dados reais, com valores menores a
fila se enche de **variantes diferentes** que devem continuar separadas
(cerveja × cerveja zero, leite desnatado × semidesnatado, IPA × Easy IPA).
Marcas diferentes nunca são sugeridas.

- **Aceitar** move o produto para o item sugerido e **trava** a decisão
  (`products.catalog_locked`): uma nova coleta do mesmo produto não a desfaz.
- **Recusar** é definitivo: o par não volta a ser sugerido.
- Uma decisão já tomada responde `409`.

A tela de revisão está no painel administrativo (`price-admin-web`), seção
"Produtos possivelmente iguais".

### Exemplo

`GET /api/catalog/items?search=nescau&min_markets=2`

```json
[
  {
    "id": "bebida-lactea-nescau-180ml-3fe26a",
    "name": "Bebida Láctea Nescau 180ml",
    "category": "LATICÍNIOS",
    "brand": "Nescau",
    "generic_key": "bebida lactea",
    "size_kind": "volume",
    "size_amount": 180,
    "size_label": "180 ml",
    "pack_count": 1,
    "sold_by_weight": false,
    "offers": [
      { "product_id": "bebida-lactea-nescau-180ml-cf9372", "market_name": "Atacadão - Primavera",
        "price": 2.29, "unit_price": 12.72, "unit_price_unit": "L", "collected_at": "2026-09-18T23:17:30Z" },
      { "product_id": "bebida-lactea-nescau-180ml-80ce8d", "market_name": "Carvalho Super - Homero Castelo Branco",
        "price": 2.79, "unit_price": 15.5, "unit_price_unit": "L", "collected_at": "2026-09-18T23:17:30Z" }
    ],
    "markets_count": 2,
    "min_price": 2.29,
    "max_price": 2.79
  }
]
```

(`offers` sai ordenado do mais barato; os itens, dos vendidos em mais
mercados para os vendidos em menos. Cada oferta traz também `product_name` e
`market_id`.)

### Preencher o catálogo de um banco que já tinha produtos

```bash
# 1. preenche os campos do parser nos produtos antigos (na pasta price-bot)
.venv/Scripts/python.exe -X utf8 scripts/backfill_parsed_fields.py

# 2. liga tudo ao catálogo (idempotente)
curl -X POST http://localhost:8080/api/catalog/match
# {"processed":361,"linked":361,"items_created":357,"suggestions":1,"skipped":2}
```

No banco de desenvolvimento, com os 2 mercados atuais, isso deu 361 produtos
em 357 itens; 4 itens são vendidos nos dois mercados e 1 caso foi para a
revisão.

### Limitações conhecidas

- O nome do item é o da **primeira oferta** que o criou (pode trazer "390gr"
  em vez de "390g"); a identidade não depende dele.
- Só produtos com campos do parser entram. A cobertura depende da lista de
  marcas do bot (`price-bot/src/nextprice/data/brands.txt`).
- Duas ofertas do **mesmo mercado** para o mesmo item aparecem as duas.
- O catálogo só junta o **mesmo** produto. "Equivalente" (outra marca do mesmo
  tipo e tamanho) é resolvido na hora da [comparação de listas](#comparação-de-listas).
- Um preço muito fora do padrão (a Maionese Heinz 390 g sai R$ 14,39 num
  mercado e R$ 27,50 no outro) ainda não gera alerta.

## Comparação de listas

`POST /api/comparisons/list` responde "quanto custa esta lista em cada mercado,
e onde cada item é mais barato?". Usa o [catálogo canônico](#catálogo-canônico),
então o mesmo produto é encontrado em todos os mercados que o vendem.

### Pedido

```json
{
  "items": [
    { "catalog_id": "bebida-lactea-nescau-180ml-3fe26a", "quantity": 6 },
    { "product_id": "arroz-branco-camil-tipo-1-5kg-a1b2c3", "quantity": 1 }
  ],
  "allow_equivalents": true,
  "lat": -5.07, "lng": -42.785, "radius_km": 10
}
```

- Cada item é um `catalog_id` **ou** o `product_id` de qualquer oferta dele
  (resolvido para o catálogo). Itens repetidos têm as quantidades somadas.
- `allow_equivalents` (padrão `false`): um mercado sem a marca pedida pode
  completar a lista com outra marca do mesmo produto. Veja abaixo.
- `lat` + `lng` (juntos) acrescentam a distância a cada mercado; com `radius_km`,
  mercados que se sabe estarem mais longe ficam de fora.

### Resposta

| Campo | O que é |
|---|---|
| `markets[]` | um por mercado, **já ranqueado** (veja abaixo) |
| `markets[].total_cost` | soma de preço × quantidade dos itens que o mercado tem |
| `markets[].is_complete` | o mercado tem **todos** os itens da lista |
| `markets[].found_items` / `total_items` | quantos itens tem / quantos foram pedidos |
| `markets[].missing[]` | itens que o mercado não vende |
| `markets[].difference` | quanto a mais que o mercado completo mais barato; **0 nos incompletos** |
| `markets[].substitutes_count` | quantas linhas usam outra marca |
| `markets[].distance_km` | distância (`null` sem `lat/lng` ou sem localização do mercado) |
| `markets[].lines[]` | uma por item: `product_name`, `brand`, `price`, `line_total`, `unit_price`, `is_cheapest`, `is_substitute` |
| `items[]` | cada item pedido: `available`, `cheapest_market_id`, `cheapest_price` |
| `best_single_market_id` | o mercado completo mais barato (vazio se nenhum é completo) |
| `best_combination` | comprar cada item onde é mais barato (veja abaixo) |
| `max_savings` / `max_savings_percent` | mais barato × mais caro **entre os completos**; 0 com menos de dois |

**Ranking.** Primeiro os mercados completos, do mais barato ao mais caro (em
empate, o mais perto); depois os incompletos, o que tem mais itens primeiro.
Um mercado que não tem metade da lista parece barato só porque falta coisa, por
isso o total dele **nunca** é comparado com o dos completos: `difference` fica
em 0 e `max_savings` só olha os completos.

**`is_cheapest`.** A linha marcada é a de menor preço daquele item entre todos
os mercados (em empate, todas as empatadas).

### Outras marcas (`allow_equivalents`)

Com a opção ligada, se um mercado não tem o produto pedido, ele pode completar
com **outra marca do mesmo produto**, em embalagem do mesmo tamanho (até 10% de
diferença: 900 ml × 1 L; nunca 2 kg × 5 kg) e com a mesma quantidade de
unidades. A linha vem com `is_substitute: true` e a marca que foi usada.

**O que conta como "o mesmo produto"** (qualquer um dos dois basta):

- **Mesmo básico** (`internal/repository/staples.go`). O nome é lido como **tipo
  + qualificadores**, e o resto (marca, embalagem, "longa vida", "com tampa") é
  ignorado:
  - arroz: branco/parboilizado/integral, tipo 1/2/3;
  - feijão: carioca/preto/fradinho/sempre verde…, tipo;
  - açúcar, sal, óleo (soja, canola…);
  - leite (integral/semi/desnatado, zero lactose), leite condensado, creme de
    leite;
  - farinha de trigo e de mandioca, flocão, fubá, macarrão (formato, com ovos);
  - molho e extrato de tomate, vinagre, água mineral (com/sem gás), papel
    higiênico (folha), detergente.

  Exemplo: "Oleo Soja Sinha 900ml" substitui "Óleo de Soja Soya 900ml", mesmo
  com o parser tendo deixado "sinha" na chave genérica. Tipo 1 pode substituir
  tipo 2 (melhor ou igual), nunca o contrário.
- **Mesma chave genérica e mesmo tamanho**, para as palavras da lista de
  `internal/repository/equivalence.go` (a regra antiga, que continua valendo).

Regras:

1. **A marca pedida sempre vence.** Se o mercado a vende, é ela que entra, mesmo
   havendo outra mais barata. Entre vários substitutos, entra o de menor preço
   por kg/L.
2. **Só para produtos em que trocar de marca é normal.** As duas listas são
   *permitidas* de propósito: o pior caso de um esquecimento é perder uma
   economia, e não trocar em silêncio a cerveja ou o vinho de alguém. Palavras
   como `chocolate`, `doce` e `cerveja` **vetam** a troca (para "chocolate ao
   leite" não virar "leite"), e cada tipo tem as suas exclusões ("leite de
   coco", "sal para churrasco"). Café continua fora: a marca importa.
3. Para permitir mais tipos, acrescente o tipo a `stapleTypes` (com as
   exclusões e qualificadores) e um caso real em `staples_test.go`.

### Melhor combinação

`best_combination` compra **cada item onde ele sai mais barato**, mesmo que isso
signifique vários mercados: `stops[]` traz, por mercado, o subtotal e as linhas.
Em empate de preço, fica no mercado que já está na rota (e, depois, no mais
perto), para não criar uma parada à toa.
`savings_vs_best_single` diz quanto isso custa a menos que o melhor mercado único
(`null` se nenhum mercado tem a lista completa).

> **O cálculo não pesa o deslocamento.** Uma combinação em 2 mercados pode
> economizar centavos (num teste real: R$ 0,79) — quem exibe deve mostrar as
> paradas e a economia juntas, para a pessoa decidir se vale a viagem.

### Erros

| Código | Quando |
|---|---|
| 400 | lista vazia; item sem `catalog_id`/`product_id`; quantidade ≤ 0; `lat` sem `lng`; `radius_km` sem posição ou ≤ 0 |
| 400 | `product_id` inexistente; `catalog_id` inexistente |
| 400 | produto **fora do catálogo** — a resposta traz `product_id` dizendo qual. Produtos sem os campos do parser (os da página da empresa) não têm como ser comparados |

`POST /api/comparisons` (o antigo, por `product_id` no mercado de origem)
continua funcionando sem alteração, mas não cruza mercados de verdade.

### Limitações

- Não considera custo de deslocamento nem tempo de viagem.
- O preço mostrado é o último coletado (`collected_at` vem em cada linha); preços
  velhos ainda não são descartados nem ponderados.
- Vale tanto quanto o catálogo: com poucos mercados coletados, poucos itens são
  vendidos em mais de um.

## Localização e distância dos mercados

`GET /api/markets?lat=&lng=` devolve `distance_km` (Haversine) e ordena do
mais perto para o mais longe. Para a distância ser real, o mercado precisa de
coordenadas reais:

- Ao cadastrar (`POST`) ou editar (`PATCH`) um mercado, o endereço é
  geocodificado pelo **OpenStreetMap/Nominatim** (`internal/geo`): rua+número,
  rua, CEP e bairro, nessa ordem, no máximo 1 requisição por segundo e com
  `User-Agent` identificado, como a política de uso pede. Só a cidade, sozinha,
  não geocodifica — travaria todos os mercados no centro.
- Coordenadas enviadas explicitamente (`latitude` + `longitude`) têm
  prioridade e dispensam o geocodificador.
- Se não for possível localizar, o mercado é salvo mesmo assim, com
  `has_location: false`: ele **continua na lista, com `distance_km: null`**, em
  vez de receber uma distância inventada. Editar o endereço tenta de novo; se
  o endereço mudar e não puder ser localizado, a localização antiga é
  descartada (ela descreveria o endereço errado).
- Um `PATCH` que reenvia o mesmo endereço não chama o Nominatim de novo.
