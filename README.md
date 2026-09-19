# price-empresa-web

Página da Empresa do Price — área estática (HTML/CSS/JS puro, sem build)
para empresas parceiras cadastrarem seus produtos e preços, que passam a
aparecer no app junto com os dados coletados automaticamente.

Não existe autocadastro: o acesso de cada empresa é criado por um admin
no Painel Administrativo (`price-admin-web`).

## Depende de

| Serviço | Porta | Para quê |
|---|---|---|
| `price-auth` | `8081` | login (`/auth/login`) |
| `price-back` | `8080` | listar mercados (autocomplete) e cadastrar produtos |

Suba os dois antes de usar a página.

## Rodar

```bash
python -m http.server 8091 --directory price-empresa-web
```

Ou via `.claude/launch.json` na raiz do projeto. Acesse
`http://localhost:8091`.

## Login

Peça a um administrador do Price para criar seu usuário e senha no
Painel Administrativo — esta página só faz login, não tem cadastro.

A tela de entrada é dividida em dois painéis, com botão de mostrar/ocultar
senha. Só contas com `role: "company"` entram; uma conta `admin` é recusada
com "Esta conta não é de uma empresa parceira" (admins usam o
`price-admin-web`). "Lembrar-me" marcado guarda a sessão no `localStorage`
(sobrevive ao fechar o navegador); desmarcado, ela vale só para a aba
(`sessionStorage`). Não há recuperação de senha por aqui — o link "esqueci a
senha" apenas orienta a falar com um administrador.

## O que a página faz

- **Cadastrar produto**: mercado (com autocomplete a partir de
  `GET /api/markets?all=true`), nome, categoria, unidade e preço. Envia
  `POST /api/products` ao `price-back`; se o produto já existe naquele
  mercado, o preço é atualizado.
- **Meus produtos cadastrados recentemente**: os 10 primeiros de
  `GET /api/products?source=company`, com mercado, categoria e preço.

Os produtos cadastrados aqui entram com `source: "company"` e aparecem no app
junto com os coletados pelo `price-bot`.

## Arquivos

| Arquivo | O que é |
|---|---|
| `index.html` | a página inteira (HTML, CSS e JS) |
| `logo-price.svg` | logo usada na tela de login e no cabeçalho |
| `favicon-32.png`, `favicon-192.png` | ícones da aba e do atalho (`apple-touch-icon`) |
