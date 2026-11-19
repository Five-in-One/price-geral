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
