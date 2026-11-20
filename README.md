# Price — repositório geral (ambiente local)

Reúne tudo o que roda **localmente** no projeto Price, cada parte na sua pasta.
O app de produção (`price-front`, que usa `https://api.pricebrasil.com`) não está aqui.

| Pasta | O que é | Porta |
|---|---|---|
| [`price-back`](price-back) | API de produtos, mercados e comparação (Go) — também tem o `docker-compose.yml` com Postgres e RabbitMQ | 8080 |
| [`price-auth`](price-auth) | API de autenticação (Go) | 8081 |
| [`price-user`](price-user) | API de usuários e listas salvas (Go) | 8082 |
| [`price-bot`](price-bot) | Coletor de preços dos supermercados (Python) | 8085 |
| [`price-admin-web`](price-admin-web) | Painel do administrador (HTML) | 8090 |
| [`price-empresa-web`](price-empresa-web) | Painel da empresa/mercado (HTML) | 8091 |
| [`price-front-local`](price-front-local) | App Flutter apontando para as APIs locais (instala como "Price Local") | — |

## Como subir tudo (Docker, recomendado)

Sobe o sistema inteiro atrás do gateway Nginx, com o `price-back` em 3 réplicas
balanceadas e as APIs protegidas por JWT. Detalhes e roteiro de testes em
[ENTREGA4.md](ENTREGA4.md).

```bash
cp .env.example .env          # defina JWT_SECRET e, para a coleta, GEMINI_API_KEY
docker compose up -d --build
```

- Gateway: <http://localhost> (único endereço exposto)
- Painel Administrativo: <http://localhost/admin/> · Página da Empresa: <http://localhost/empresa/>
- App: `cd price-front-local && flutter run --dart-define=API_HOST=10.0.2.2` (emulador Android; num celular use o IP do computador)
- Demonstração dos critérios da Entrega 4: `powershell -ExecutionPolicy Bypass -File scripts\entrega4-demo.ps1`

## Entrega 4 — passo a passo dos testes (para o professor)

Roteiro para conferir cada critério de aceitação da Entrega 4 pelo
Postman/Insomnia ou pelo terminal. Os comandos de terminal aparecem em duas
versões: PowerShell (Windows) e bash (Linux/macOS/Git Bash). No PowerShell use
`curl.exe`, porque `curl` sozinho é outro comando lá.

### 0. Subir o sistema

Pré-requisitos: Git e Docker Desktop (ou Docker Engine com Compose), com a
porta 80 livre.

```bash
git clone https://github.com/Five-in-One/price-geral.git
cd price-geral
cp .env.example .env            # PowerShell: Copy-Item .env.example .env
docker compose up -d --build    # a 1ª vez demora alguns minutos (compila as imagens)
docker compose ps
```

Resultado esperado em `docker compose ps`:
- `gateway`, `price-auth`, `price-user`, `postgres`, `rabbitmq` e `price-bot-worker` no ar;
- três contêineres do `price-back`: `price-price-back-1`, `-2` e `-3`;
- só o `gateway` com porta publicada (`0.0.0.0:80->80/tcp`). Os outros mostram apenas a porta interna, sem `->`.

> O `.env.example` já funciona como está. A `GEMINI_API_KEY` só é necessária
> para a coleta de preços do bot, que não faz parte destes testes.

### Parte 1 — Segurança (JWT)

A rota protegida usada nos testes é `GET http://localhost/api/catalog/suggestions`:
fica no `price-back` (Resource Server) e exige papel `admin`.

**1. Requisição sem token → 401 Unauthorized**

```powershell
curl.exe -i http://localhost/api/catalog/suggestions
```

```bash
curl -i http://localhost/api/catalog/suggestions
```

Resposta: `HTTP/1.1 401 Unauthorized` e `{"error":"missing bearer token"}`.

**2. Login no Authorization Server e captura do JWT**

Crie um usuário comum (papel `user`) e faça login. No Postman:
`POST http://localhost/api/auth/signup` e depois `POST http://localhost/api/auth/login`,
com o corpo JSON `{"username":"professor","password":"Senha12345"}`.

```powershell
# PowerShell: cria a conta, faz login e guarda o JWT em $token
$body  = @{ username = "professor"; password = "Senha12345" } | ConvertTo-Json
Invoke-RestMethod -Method Post -Uri http://localhost/api/auth/signup -ContentType "application/json" -Body $body
$token = (Invoke-RestMethod -Method Post -Uri http://localhost/api/auth/login -ContentType "application/json" -Body $body).access_token
$token
```

```bash
# bash
curl -X POST http://localhost/api/auth/signup -H "Content-Type: application/json" -d '{"username":"professor","password":"Senha12345"}'
curl -X POST http://localhost/api/auth/login  -H "Content-Type: application/json" -d '{"username":"professor","password":"Senha12345"}'
```

A resposta do login traz `access_token`, que é o JWT. Copie o valor. Se a
conta já existir, o signup responde 409, o que não atrapalha o login.

Cole o token no <https://jwt.io>. O payload tem só a identificação do usuário,
o papel e as datas, sem senha nem dados pessoais:

```json
{ "user_id": "…", "username": "professor", "role": "user", "sub": "…", "iat": …, "exp": … }
```

Com o token verdadeiro, a rota responde **403 Forbidden**: a assinatura é
válida, mas o papel `user` não tem permissão. Isso mostra que o token é lido e
conferido.

```powershell
curl.exe -i http://localhost/api/catalog/suggestions -H "Authorization: Bearer $token"
```

```bash
curl -i http://localhost/api/catalog/suggestions -H "Authorization: Bearer COLE_O_TOKEN_AQUI"
```

**3. Adulterar o token no jwt.io**

No jwt.io, edite o payload e troque `"role": "user"` por `"role": "admin"`.
Copie o token que o site gera. Ele não conhece o segredo do servidor, então a
assinatura não corresponde ao novo conteúdo.

**4. Requisição com o token adulterado → recusada**

```powershell
curl.exe -i http://localhost/api/catalog/suggestions -H "Authorization: Bearer COLE_O_TOKEN_ADULTERADO"
```

```bash
curl -i http://localhost/api/catalog/suggestions -H "Authorization: Bearer COLE_O_TOKEN_ADULTERADO"
```

Resposta: `HTTP/1.1 401 Unauthorized` e `{"error":"invalid or expired token"}`.
O microsserviço recalcula o HMAC-SHA256 com o segredo compartilhado, que não
bate com a assinatura, e recusa a requisição.

> Outros Resource Servers: o `price-user` responde do mesmo jeito em
> `GET http://localhost/api/shopping-lists`. Sem token ou com token adulterado
> dá 401, e com o token verdadeiro dá 200.

### Parte 2 — Escalabilidade (Nginx + 3 réplicas)

**5. 5 a 10 requisições seguidas → tráfego distribuído entre os 3 contêineres**

Pelo Postman/Insomnia, envie várias vezes `GET http://localhost/api/markets`
e observe os cabeçalhos da resposta:
- `X-Instance`: hostname (ID) do contêiner do `price-back` que atendeu;
- `X-Upstream`: IP:porta da réplica.

Os valores se alternam em rodízio (Round Robin) entre as três réplicas.

Pelo terminal:

```powershell
# PowerShell
1..9 | ForEach-Object { (Invoke-WebRequest http://localhost/api/markets -UseBasicParsing).Headers["X-Instance"] }
```

```bash
# bash
for i in $(seq 1 9); do curl -s -o /dev/null -D - http://localhost/api/markets | grep -i x-instance; done
```

Nos logs do Nginx, cada linha mostra `upstream=` (a réplica que respondeu) e
`instance=` (o contêiner):

```bash
docker compose logs --tail 10 gateway
```

Para relacionar o ID em `instance=` com o nome do contêiner:

```bash
docker ps --filter name=price-back --format "table {{.ID}}\t{{.Names}}"
```

**6. Derrubar um contêiner do backend em tempo real**

```bash
docker stop price-price-back-2
```

**7. Novas requisições → o sistema continua respondendo 200 OK**

Repita as requisições do passo 5. Todas respondem **200**, alternando entre as
duas réplicas que sobraram.

Se uma requisição cair na réplica derrubada antes de o Nginx percebê-la, ele a
repassa na hora para outra réplica (`proxy_next_upstream`), e o cliente recebe
200 do mesmo jeito. No log aparece algo como
`upstream=172.x.x.8:8080, 172.x.x.6:8080` (tentou a caída, atendeu a seguinte).

Para religar a réplica, que volta ao rodízio sozinha:

```bash
docker start price-price-back-2
```

### Atalhos

- **Coleção do Postman** com os testes prontos: importe
  [scripts/entrega4.postman_collection.json](scripts/entrega4.postman_collection.json)
  e execute na ordem. O login guarda o token e gera a versão adulterada.
- **Script que roda tudo de uma vez** (Windows):
  `powershell -ExecutionPolicy Bypass -File scripts\entrega4-demo.ps1`
- Onde está cada requisito no código: [ENTREGA4.md](ENTREGA4.md).

### Encerrar

```bash
docker compose down        # para tudo (mantém os dados)
docker compose down -v     # para tudo e apaga os dados
```

## Como subir cada parte separada (desenvolvimento)

1. **Infra** (Postgres :5432 e RabbitMQ :5672 / painel :15672), uma vez só:
   ```bash
   cd price-back && docker compose up -d
   ```
2. **Configuração**: em `price-back`, `price-auth`, `price-user` e `price-bot`,
   copie `.env.example` para `.env` e ajuste (o `JWT_SECRET` precisa ser o mesmo
   em `price-auth`, `price-user` e `price-back`; o `price-bot` precisa de
   `GEMINI_API_KEY` e de `BOT_USERNAME`/`BOT_PASSWORD` para obter o JWT).
3. **APIs** (um terminal cada):
   ```bash
   cd price-back && go run ./cmd/api
   cd price-auth && go run ./cmd/api
   cd price-user && go run ./cmd/api
   ```
4. **Bot**:
   ```bash
   cd price-bot
   python -m venv .venv && source .venv/bin/activate   # Windows: .venv\Scripts\activate
   pip install -e ".[dev]"
   playwright install chromium   # navegador usado na coleta (sem ele a coleta falha)
   python main.py worker         # consome os pedidos de coleta do Painel Admin (RabbitMQ)
   ```
   Opcional: `python -m nextprice.collect_api` sobe a API de coleta direta na 8085.
5. **Painéis** (a partir da raiz deste repositório):
   ```bash
   python -m http.server 8090 --directory price-admin-web
   python -m http.server 8091 --directory price-empresa-web
   ```
6. **App**:
   ```bash
   cd price-front-local
   flutter pub get
   flutter run --dart-define=API_HOST=<ip do computador> --dart-define=API_GATEWAY=false   # sem o gateway: uma porta por serviço
   ```

Os detalhes de cada parte estão no README da própria pasta.
