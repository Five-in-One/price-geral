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
Sobe em `:8081` por padrão. Defina `JWT_SECRET` em produção — sem ele, usa um
valor de desenvolvimento fixo.

| Variável | Padrão | Para quê |
|---|---|---|
| `DATABASE_URL` | `postgres://postgres:postgres@localhost:5432/price_auth_db?sslmode=disable` | conexão com o Postgres |
| `PORT` | `8081` | porta HTTP |
| `JWT_SECRET` | `dev-secret-change-me` | segredo que assina os tokens (HS256) |

Há um `.env.example` com esses valores.

## Rodar os testes

Com o Postgres do `price-back` já rodando:

```bash
go test ./...
```

## Rotas

| Método | Rota | Descrição |
|---|---|---|
| GET | `/health` | healthcheck |
| POST | `/auth/register` | `{username,password}` → cria a conta como `role: "company"` e retorna `access_token` + usuário |
| POST | `/auth/signup` | `{username,password}` → cria a conta como `role: "user"` (quem usa o app) e retorna `access_token` + usuário. É o cadastro do app Flutter |
| POST | `/auth/login` | `{username,password}` → retorna `access_token` + usuário (com `role`) |
| GET | `/auth/me` | `Authorization: Bearer <token>` → retorna o usuário autenticado |
| POST | `/auth/admin/users` | **admin only** — `{username,password,role}` → cria uma conta com o `role` pedido (`admin`, `company` ou `user`) |

`username` é normalizado para minúsculas e precisa ter entre 3 e 32
caracteres, sem espaços. O token é um JWT HS256 (`user_id`, `username`,
`role`, expira em 24h) assinado com `JWT_SECRET`. Outros serviços que
compartilharem o mesmo segredo podem validar o token localmente sem chamar
esta API.

## Roles

Existem três: `admin` (equipe interna, usa o Painel Administrativo),
`company` (empresa parceira, usa a Página da Empresa) e `user` (pessoa que
usa o app; guarda listas de compras no `price-user`). Ninguém vira `admin`
se autocadastrando — `/auth/register` sempre cria `company` e `/auth/signup`
sempre cria `user`, ignorando qualquer `role` enviada no corpo. A única forma de criar uma conta `admin`
é outro admin chamar `POST /auth/admin/users`.

Para existir um primeiro admin, a migration `0003_seed_admin.sql` já cria:

```
username: admin
senha: TrocarSenha123!
```

Troque essa senha (ou crie outro admin e pare de usar essa conta) antes de
qualquer uso além de desenvolvimento local.
