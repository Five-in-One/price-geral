# price-back

API de Produtos e Preços do Price (Go + chi + pgx), com PostgreSQL rodando
em Docker. Cuida só de mercados, produtos e comparação de preços — login
fica em `price-auth` e histórico/economia do usuário em `price-user`
(ambos serviços separados que compartilham este mesmo Postgres).

## Subir o banco

```bash
docker compose up -d
```

Isso sobe o Postgres compartilhado pelos três serviços (`price-back`,
`price-auth`, `price-user`) e cria os bancos de cada um: `price_db` /
`price_db_test` (este serviço), `price_auth_db` / `price_auth_db_test` e
`price_user_db` / `price_user_db_test`.

## Rodar a API

```bash
go run ./cmd/api
```

As migrations em `migrations/*.sql` são aplicadas automaticamente na
inicialização (schema + seed com os mesmos dados de `mock_data.dart` do
front). A API sobe em `:8080` por padrão (`PORT` no ambiente para mudar).

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
| GET | `/api/products?category=&search=&sort=price_asc` | busca/filtra produtos |
| POST | `/api/comparisons` | compara o custo de uma lista `{items:[{product_id,quantity}]}` entre todos os mercados |

Histórico de economia (`/api/savings-history`, `/api/economy/summary`)
mudou para o serviço `price-user` — veja `../price-user/README.md`.
