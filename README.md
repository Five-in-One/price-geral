# price-auth

API de autenticação do Price (Go + chi + pgx), separada da API de produtos e
preços (`price-back`). Cuida só de cadastro de credenciais, login e emissão
de token — perfil, favoritos e histórico do usuário ficam para uma futura
API de Registro de Usuário.

## Banco de dados

Usa o **mesmo Postgres em Docker do `price-back`**, mas com bancos próprios
(`price_auth_db` e `price_auth_db_test`), sem nenhuma tabela compartilhada
com `price_db`. Não sobe um Postgres próprio: suba o do `price-back` antes
de rodar esta API.

```bash
cd ../price-back
docker compose up -d
```

Em uma instalação nova, os bancos `price_auth_db`/`price_auth_db_test` são
criados automaticamente pelo script em
`../price-back/docker/initdb/02-create-auth-dbs.sql`.

## Rodar a API

```bash
go run ./cmd/api
```

Migrations em `migrations/*.sql` aplicam automaticamente na inicialização.
Sobe em `:8081` por padrão (`PORT` no ambiente para mudar). Defina
`JWT_SECRET` em produção — sem ele, usa um valor de desenvolvimento fixo.

## Rodar os testes

Com o Postgres do `price-back` já rodando:

```bash
go test ./...
```

## Rotas

| Método | Rota | Descrição |
|---|---|---|
| GET | `/health` | healthcheck |
| POST | `/auth/register` | `{email,password}` → cria a conta e retorna `access_token` + usuário |
| POST | `/auth/login` | `{email,password}` → retorna `access_token` + usuário |
| GET | `/auth/me` | `Authorization: Bearer <token>` → retorna o usuário autenticado |

O token é um JWT HS256 (`user_id`, `email`, expira em 24h) assinado com
`JWT_SECRET`. Outros serviços que compartilharem o mesmo segredo podem
validar o token localmente sem chamar esta API.
