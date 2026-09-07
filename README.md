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
