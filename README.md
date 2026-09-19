# price-admin-web

Painel Administrativo do Price — página estática (HTML/CSS/JS puro, sem
build) para a equipe interna: login, cadastro de novas contas (admin ou
empresa parceira), coleta de produtos via IA a partir do link de um
mercado, e listagem/busca de mercados e produtos cadastrados.

Não existe autocadastro aqui: contas só são criadas por um admin já
logado, através da seção "Novo usuário".

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

## Telas

### Editar mercado (endereço)

O botão "Editar" de um mercado abre um modal com nome, iniciais, cor, **endereço**
(CEP, logradouro, número, bairro, cidade, UF e complemento) e os links do site.
Digitar o CEP preenche o resto pelo ViaCEP.

Ao salvar com o endereço alterado, o `price-back` localiza o novo endereço no
mapa (é o que faz a distância aparecer no app). O modal mostra a situação —
"Localização no mapa: encontrada / não encontrada" — e, se o endereço novo não
puder ser localizado, avisa e **continua aberto** para você corrigir. Mercados
sem localização aparecem na tabela com a marca "sem localização" e não mostram
distância no app.

O modal é responsivo: em telas estreitas os campos empilham e o conteúdo rola
por dentro, e as tabelas rolam na horizontal em vez de alargar a página.

### Produtos possivelmente iguais

Seção no fim da página, alimentada pela fila de revisão do catálogo canônico
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
