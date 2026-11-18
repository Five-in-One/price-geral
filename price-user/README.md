# price-user

API de Registro de Usuário do Price (Go + chi + pgx) — o que pertence a cada
pessoa: as listas de compras que ela salvou e o histórico (comparações
passadas, resumo de economia). Separada da API de Produtos e Preços
(`price-back`) e da API de Autenticação (`price-auth`), que emite os tokens
que este serviço valida. Perfil e favoritos ainda não existem aqui.

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
| `JWT_SECRET` | `dev-secret-change-me` | segredo que assina os tokens do `price-auth` — **precisa ser o mesmo valor** do `price-auth`, senão as rotas de listas salvas respondem `401` |

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
| GET | `/api/shopping-lists` | **login** — as listas salvas de quem está logado, da mais recente para a mais antiga |
| POST | `/api/shopping-lists` | **login** — `{name, items:[{product:{…}, quantity}]}` salva a lista. Se a pessoa já tem uma lista com esse nome (sem diferenciar maiúsculas), ela é substituída (`200`); senão é criada (`201`) |
| DELETE | `/api/shopping-lists/{id}` | **login** — apaga uma lista da própria pessoa (`204`; a de outra pessoa dá `404`) |

## Listas salvas

As rotas de `/api/shopping-lists` exigem `Authorization: Bearer <token>` com um
token emitido pelo `price-auth` (`/auth/signup` ou `/auth/login`); o serviço o
valida localmente com o `JWT_SECRET` e usa o `user_id` do token — cada pessoa só
enxerga e apaga as próprias listas. Os itens ficam guardados exatamente como o
app os enviou (uma cópia do produto + a quantidade), então uma lista salva
recupera os preços da época em que foi salva; a comparação entre mercados é
sempre recalculada pelo `price-back` na hora. Limites: nome de 1 a 60
caracteres, 1 a 200 itens por lista e 50 listas por pessoa (`409` ao passar).
Rotas de economia/histórico continuam abertas, como antes.
