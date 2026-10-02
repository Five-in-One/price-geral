# price-admin-web

Painel Administrativo do Price — página estática (HTML/CSS/JS puro, sem
build) para a equipe interna: login, cadastro de novas contas (admin ou
empresa parceira), coleta de produtos via IA a partir do link de um
mercado, e listagem/busca de mercados e produtos cadastrados.

Não existe autocadastro aqui: contas só são criadas por um admin já
logado, na página **Usuários**.

Ao cadastrar uma **empresa parceira**, o formulário pede também o mercado
dela e o endereço (o CEP preenche logradouro, bairro, cidade e UF via
ViaCEP). O painel salva o mercado no `price-back` (`POST /api/markets`,
`source: "company"`), que localiza o endereço no mapa, e cria a conta com
`market_id` apontando para ele. Assim o app mostra o endereço ("Ver
endereço", "Abrir no mapa") e a distância até o mercado, e a Página da
Empresa já usa esse mercado. Se já existir um mercado com o mesmo nome,
a conta é ligada a ele e o endereço é atualizado. Quando o endereço não
é localizado no mapa, o painel avisa: o app mostra o endereço sem a
distância, e dá para corrigir em Mercados → Editar.

## Depende de

| Serviço | Porta | Para quê |
|---|---|---|
| `price-auth` | `8081` | login (`/auth/login`) e criação de contas (`/auth/admin/users`) |
| `price-back` | `8080` | listar mercados e produtos cadastrados |
| `price-bot` (`collect_api.py`) | `8085` | coleta sob demanda de produtos via IA |

Suba os três antes de usar o painel por completo (a página funciona sem
o `price-bot`, mas o botão "Coletar com IA" fica indisponível).

## Rodar

```bash
python -m http.server 8090 --directory price-admin-web
```

Ou via `.claude/launch.json` na raiz do projeto. Acesse
`http://localhost:8090`.

## Login

Usuário: `admin` — senha: `TrocarSenha123!` (conta seedada pela migration
`0003_seed_admin.sql` do `price-auth`). Troque essa senha ou crie outro
admin antes de qualquer uso além de desenvolvimento local.

## Visual e navegação

Depois do login, cada funcionalidade é uma página, escolhida no menu do
cabeçalho (o endereço muda para `#mercados`, `#produtos`, `#catalogo`,
`#coleta` ou `#usuarios`, então dá para favoritar ou recarregar):

| Página | O que tem |
|---|---|
| **Mercados** | tabela com busca, cor, links, origem e as ações Recarregar, Editar e Excluir |
| **Produtos** | tabela com busca e filtros por mercado, categoria e origem; Editar e Excluir |
| **Catálogo** | fila de "produtos possivelmente iguais" e o botão Reprocessar catálogo; o menu mostra quantos estão pendentes |
| **Coleta com IA** | coleta sob demanda a partir do link do site, com endereço opcional |
| **Usuários** | cadastro de empresa parceira (com mercado e endereço) ou administrador |

Mesma linguagem visual da Página da Empresa, na identidade azul do Price:
banner no topo de cada página (com números como "4 mercados · 4 no mapa"),
botões em pílula, painéis arredondados, categorias e origens em etiquetas
coloridas e ações com ícones. No celular as linhas das tabelas viram
cartões e o menu rola para o lado.

**Modo escuro**: switch no cabeçalho, só nas páginas de dentro — a tela de
login é sempre clara. Sem escolha salva, segue o tema do sistema; a escolha
fica no `localStorage` (`price_admin_theme`).

A tela de login tem o formulário num cartão branco e, à direita, um painel
azul com o logo (some em telas estreitas). Usuário ou senha errados mostram
"Usuário ou senha incorretos.".

## Telas

### Editar mercado (endereço)

O botão "Editar" de um mercado abre um modal com nome, iniciais, cor, **endereço**
(CEP, logradouro, número, bairro, cidade, UF e complemento) e os links do site.
Digitar o CEP preenche o resto pelo ViaCEP.

Na tabela de mercados a coluna "Links" mostra só a quantidade ("60 links"). As
URLs ficam no modal, numa lista com rolagem própria e um campo "Pesquisar nos
links..." que filtra sem diferenciar maiúsculas de minúsculas ("11 de 60 links");
copiar e remover continuam valendo para o link certo mesmo com a lista filtrada.

Ao salvar com o endereço alterado, o `price-back` localiza o novo endereço no
mapa (é o que faz a distância aparecer no app). O modal mostra a situação —
"Localização no mapa: encontrada / não encontrada" — e, se o endereço novo não
puder ser localizado, avisa e **continua aberto** para você corrigir. Mercados
sem localização aparecem na tabela com a marca "sem localização" e não mostram
distância no app.

O modal é responsivo: em telas estreitas os campos empilham e o conteúdo rola
por dentro.

### Produtos possivelmente iguais

Página **Catálogo**, alimentada pela fila de revisão do catálogo canônico
(`GET /api/catalog/suggestions` do `price-back`). Cada cartão mostra o cadastro
duvidoso ao lado do item do catálogo com o qual ele pode ser o mesmo produto —
com mercado e preço de cada lado — e o motivo da dúvida (falta a marca num dos
nomes, ou nomes quase idênticos).

- **São o mesmo produto**: junta os dois no catálogo e trava a decisão (uma nova
  coleta não a desfaz).
- **Não são o mesmo**: o par nunca mais é sugerido.
- **Reprocessar catálogo**: liga ao catálogo os produtos que ainda não estão
  nele (`POST /api/catalog/match`). É seguro repetir.

Só casos duvidosos chegam aqui; produtos com a mesma marca, o mesmo nome e o
mesmo tamanho já são ligados automaticamente. Como o catálogo funciona:
`../price-back/README.md`, seção "Catálogo canônico".
