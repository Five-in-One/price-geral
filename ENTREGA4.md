# Entrega 4 — Segurança (JWT) e Escalabilidade (Nginx)

Este documento mostra onde está cada requisito da Entrega 4 e como verificar
cada critério de aceitação.

## Como subir

```bash
cp .env.example .env        # defina JWT_SECRET (e GEMINI_API_KEY, se for usar a coleta do bot)
docker compose up -d --build
```

Só o **gateway (Nginx)** publica porta na máquina: <http://localhost>.

| Endereço | Vai para |
|---|---|
| `http://localhost/api/auth/*` | `price-auth`: Authorization Server (login, cadastro, emissão do JWT) |
| `http://localhost/api/shopping-lists`, `/api/economy`, `/api/savings-history` | `price-user`: Resource Server |
| `http://localhost/api/*` (o resto) | `price-back`: Resource Server, **3 réplicas** com Round Robin |
| `http://localhost/admin/` | Painel Administrativo |
| `http://localhost/empresa/` | Página da Empresa |

```text
                     ┌──────────────── rede interna price-net ──────────────────┐
 cliente ──:80──▶ Nginx ──/api/auth/──▶ price-auth ──┐                          │
 (Postman, app,     │   ──/api/shopping-lists──▶ price-user ──┤                 │
  painéis)          │   ──/api/*── Round Robin ──▶ price-back #1 ├──▶ Postgres  │
                    │                          ├─▶ price-back #2 │              │
                    │                          └─▶ price-back #3 ┴──▶ RabbitMQ ◀── price-bot-worker
                     └───────────────────────────────────────────────────────────┘
```

## Requisitos de segurança

**Authorization Server: `price-auth`.** `POST /api/auth/login` confere usuário
e senha (hash bcrypt no banco `price_auth_db`) e devolve um JWT **HS256**
([price-auth/internal/auth/jwt.go](price-auth/internal/auth/jwt.go)).

**Payload do token:** contém só identificação, permissão e validade. Senha,
e-mail e outros dados pessoais não vão no token.

```json
{ "user_id": "…", "username": "usuario-demo", "role": "user", "sub": "…", "iat": 1763500000, "exp": 1763586400 }
```

**Resource Servers:**
- O `price-back` ([internal/httpserver/auth.go](price-back/internal/httpserver/auth.go)) e o `price-user` ([internal/httpserver/middleware.go](price-user/internal/httpserver/middleware.go)) interceptam a requisição e leem `Authorization: Bearer <token>`.
- Cada um confere a assinatura HMAC-SHA256 com o segredo compartilhado (`JWT_SECRET`). Só aceita `alg=HS256`, então `alg=none` é recusado, e exige `exp`.
- Respostas:
  - **401** sem token, com token adulterado, assinado com outro segredo ou expirado;
  - **403** com token válido, mas papel sem permissão.

| Rotas do `price-back` | Quem acessa |
|---|---|
| `GET /api/markets`, `GET /api/products*`, `POST /api/comparisons*`, `GET /api/catalog/items*` | Público: o app funciona sem conta |
| `POST/PATCH/DELETE /api/products`, `PATCH /api/markets/{id}` | `admin` ou `company` |
| `POST/DELETE /api/markets`, links, coleta (`/api/admin/collect`, `/api/markets/{id}/collect`, `/api/collect/jobs/{id}`), revisão do catálogo (`/api/catalog/suggestions*`, `/api/catalog/match`) | só `admin` |

Todas as rotas de negócio do `price-user` exigem token.

O worker do bot também é cliente do Authorization Server: ele faz login com a
conta configurada em `BOT_USERNAME`/`BOT_PASSWORD` e manda o Bearer ao gravar
produtos ([price-bot/src/nextprice/price_back_auth.py](price-bot/src/nextprice/price_back_auth.py)).

## Requisitos de escalabilidade

**Nginx oficial no compose** ([docker-compose.yml](docker-compose.yml), serviço
`gateway`, imagem `nginx:1.28-alpine`). Ele é o **único** serviço com `ports:`.
Postgres, RabbitMQ, as APIs e o worker não publicam porta nenhuma.

**Roteamento por caminho** em [gateway/nginx.conf](gateway/nginx.conf), com
preservação de contexto:
- `X-Real-IP`
- `X-Forwarded-For`
- `X-Forwarded-Proto`
- `X-Forwarded-Host`
- `Host`

**Escalabilidade horizontal:**
- `price-back` tem `deploy.replicas: 3`.
- O bloco `upstream price_back` usa **Round Robin**, o algoritmo padrão do Nginx.
- `resolve` mantém a lista de réplicas atualizada pelo DNS do Docker.
- `max_fails=1 fail_timeout=10s` tira do rodízio uma réplica que falhou.
- `proxy_next_upstream error timeout http_502 http_503 http_504` reenvia a requisição para outra réplica quando a escolhida está fora do ar.

**Rastreabilidade:** cada réplica devolve o próprio hostname no cabeçalho
`X-Instance`. O Nginx registra no log o `upstream=` (IP:porta) e o `instance=`
de cada requisição, e devolve `X-Upstream` na resposta.

## Verificando os critérios de aceitação

### Opção 1: script

```powershell
powershell -ExecutionPolicy Bypass -File scripts\entrega4-demo.ps1
```

O script executa todos os passos e imprime o resultado de cada um:
1. Rota protegida sem token: espera 401.
2. Login no Authorization Server e exibição do payload do JWT.
3. Token verdadeiro de usuário: espera 403.
4. Token com `role` trocada de `user` para `admin`: espera 401.
5. 9 requisições pelo Nginx, mostrando a réplica que atendeu cada uma.
6. Derruba uma réplica.
7. Mais 6 requisições: todas devem dar 200.
8. Religa a réplica.

### Opção 2: Postman/Insomnia

Importe [scripts/entrega4.postman_collection.json](scripts/entrega4.postman_collection.json)
e rode as requisições na ordem. O login guarda o token, e o mesmo script já
monta a versão adulterada, com `role=admin`, para o passo 4.

### Opção 3: manual, com o jwt.io

1. `GET http://localhost/api/catalog/suggestions` sem cabeçalho: espera **401**.
2. `POST http://localhost/api/auth/signup` e depois `POST http://localhost/api/auth/login` com `{"username":"usuario-demo","password":"SenhaDemo123!"}`. Copie o `access_token`.
3. Cole o token no jwt.io e troque `"role": "user"` por `"role": "admin"` no payload. O jwt.io gera um token novo com a assinatura antiga.
4. `GET http://localhost/api/catalog/suggestions` com `Authorization: Bearer <token adulterado>`: espera **401** `invalid or expired token`.

**Balanceamento e queda de réplica:**

```bash
# 5 a 10 requisições e os logs mostrando as 3 réplicas
for i in $(seq 1 9); do curl -s -o /dev/null -D - http://localhost/api/markets | grep -i x-instance; done
docker compose logs gateway | tail -9

# derrubar uma réplica em tempo real
docker compose ps price-back                 # lista price-price-back-1, -2 e -3
docker stop price-price-back-2
for i in $(seq 1 6); do curl -s -o /dev/null -w "%{http_code}\n" http://localhost/api/markets; done   # todas 200
docker start price-price-back-2
```

## Clientes

- **Painéis:** em `/admin/` e `/empresa/` usam o próprio gateway (mesma origem) e mandam o JWT da sessão em todas as chamadas ao `price-back`.
- **App Flutter (`price-front-local`):** por padrão fala com o gateway na porta 80. Rode com `--dart-define=API_HOST=10.0.2.2` no emulador Android ou com o IP do computador num celular real. Com `--dart-define=API_GATEWAY=false`, volta a usar uma porta por serviço (desenvolvimento com `go run`).
