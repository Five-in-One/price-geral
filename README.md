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
