# price-user

API de Registro de Usuário do Price (Go + chi + pgx) — histórico do
usuário (comparações passadas, resumo de economia). Separada da API de
Produtos e Preços (`price-back`) e da API de Autenticação (`price-auth`).
Cadastro de conta, perfil e favoritos ainda não existem aqui — hoje o
serviço só serve os dados de economia que o front já consumia.

## Banco de dados

Usa o **mesmo Postgres em Docker do `price-back`**, com bancos próprios
(`price_user_db` e `price_user_db_test`), sem nenhuma tabela compartilhada
com `price_db` ou `price_auth_db`. Não sobe um Postgres próprio: suba o do
`price-back` antes de rodar esta API.

```bash
cd ../price-back
docker compose up -d
```

Em uma instalação nova, os bancos `price_user_db`/`price_user_db_test` são
criados automaticamente pelo script em
`../price-back/docker/initdb/03-create-user-dbs.sql`.

## Rodar a API

```bash
go run ./cmd/api
```

Migrations em `migrations/*.sql` aplicam automaticamente na inicialização.
Sobe em `:8082` por padrão.

| Variável | Padrão | Para quê |
|---|---|---|
| `DATABASE_URL` | `postgres://postgres:postgres@localhost:5432/price_user_db?sslmode=disable` | conexão com o Postgres |
| `PORT` | `8082` | porta HTTP |

Há um `.env.example` com esses valores.

Fora dos testes o banco começa **vazio**: a migration `0002_seed_data.sql` não
insere mais dados de demonstração. Enquanto não houver histórico real,
`/api/savings-history` devolve `[]` e `/api/economy/summary` devolve `404`
(`no economy summary configured`). Os dados de exemplo usados pelos testes
ficam em `internal/testutil/db.go`.

## Rodar os testes

Com o Postgres do `price-back` já rodando:

```bash
go test ./...
```

## Rotas

| Método | Rota | Descrição |
|---|---|---|
| GET | `/health` | healthcheck |
| GET | `/api/savings-history` | histórico de comparações/economias |
| GET | `/api/economy/summary` | totais do mês + série mensal do gráfico |
