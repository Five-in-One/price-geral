# price-empresa-web

Página da Empresa do Price — área estática (HTML/CSS/JS puro, sem build)
para empresas parceiras cadastrarem seus produtos e preços, que passam a
aparecer no app junto com os dados coletados automaticamente.

Não existe autocadastro: o acesso de cada empresa é criado por um admin
no Painel Administrativo (`price-admin-web`).

## Depende de

| Serviço | Porta | Para quê |
|---|---|---|
| `price-auth` | `8081` | login (`/auth/login`) e dados da conta (`/auth/me`) |
| `price-back` | `8080` | mercado da empresa (nome e endereço) e seus produtos |

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

A tela de entrada (sempre clara) tem o formulário num cartão branco sobre
fundo com formas amarelas e, à direita, um painel amarelo com o logo e as
vantagens de ser parceiro (some em telas estreitas). Os campos têm ícone e
o botão de mostrar/ocultar senha é só o ícone: cinza, dourado ao passar o
mouse. Usuário ou senha errados mostram "Usuário ou senha incorretos.". Só contas com
`role: "company"` entram; uma conta `admin` é recusada com "Esta conta não
é de uma empresa parceira" (admins usam o `price-admin-web`). "Lembrar-me"
marcado guarda a sessão no `localStorage` (sobrevive ao fechar o
navegador); desmarcado, ela vale só para a aba (`sessionStorage`). Ao abrir,
a página atualiza a conta por `GET /auth/me` — assim, se um admin ligar a
conta a um mercado depois, não é preciso sair e entrar de novo. Não há
recuperação de senha por aqui — o link "esqueci a senha" apenas orienta a
falar com um administrador.

## O que a página faz

Depois do login, o cabeçalho tem um menu com duas páginas (`#produtos` e
`#perfil`) e o switch de modo escuro.

- **Produtos** (`#produtos`): o nome do mercado da empresa como título, com
  o endereço que o app mostra logo abaixo.
  - *Novo produto*: nome, categoria, unidade e preço → `POST /api/products`
    com o `market_id` da conta; se o produto já existe naquele mercado, o
    preço é atualizado.
  - *Produtos cadastrados*: tabela com os produtos do mercado
    (`GET /api/products?market_id=…`), com busca por nome, **Editar**
    (nome, categoria, unidade e preço → `PATCH /api/products/{id}`) e
    **Excluir** (`DELETE /api/products/{id}`, com confirmação).
- **Perfil** (`#perfil`): nome e endereço do mercado, que aparecem no app.
  O CEP preenche logradouro, bairro, cidade e UF via ViaCEP. Salvar envia
  `PATCH /api/markets/{id}` ao `price-back`, que localiza o endereço no
  mapa; o app passa a mostrar o nome, o endereço e a distância novos. Se o
  endereço não for encontrado no mapa, ele é salvo mesmo assim (o app
  mostra sem a distância) e a página avisa para conferir número e CEP. Ao
  lado, uma prévia de como o card do mercado aparece no app, atualizada
  enquanto se digita.
- **Visual**: identidade amarela da tela de login — botões em pílula,
  banner no topo de cada página com o logo e ícones flutuantes, painéis
  arredondados, categorias em etiquetas coloridas. No celular, as linhas da
  tabela viram cartões.
- **Modo escuro**: switch no cabeçalho, só nas páginas de dentro — a tela
  de login é sempre clara. Sem escolha salva, segue o tema do sistema; a
  escolha fica no `localStorage` (`price_empresa_theme`).

**Contas sem mercado ligado** (criadas antes do cadastro com endereço no
Painel Administrativo): a página avisa que a conta não está ligada a um
mercado; o novo produto pede o nome do mercado (com autocomplete de
`GET /api/markets?all=true`), a tabela lista os produtos de empresas
(`?source=company`) sem editar/excluir, e o Perfil pede para falar com um
administrador.

Os produtos cadastrados aqui entram com `source: "company"` e aparecem no app
junto com os coletados pelo `price-bot`.

## Arquivos

| Arquivo | O que é |
|---|---|
| `index.html` | a página inteira (HTML, CSS e JS) |
| `logo-price.svg` | logo usada na tela de login e no cabeçalho |
| `favicon-32.png`, `favicon-192.png` | ícones da aba e do atalho (`apple-touch-icon`) |
