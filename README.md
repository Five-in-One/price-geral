# Price (local)

> **Cópia do `price-front` que usa os servidores locais**, não a API de
> produção. Ela fala com `price-back` (:8080), `price-auth` (:8081) e
> `price-user` (:8082) pelo IP de rede do computador (`192.168.18.138`).
> Se o IP mudar, rode com `--dart-define=API_HOST=<novo ip>`
> (`10.0.2.2` no emulador Android, `localhost` no navegador/desktop).
>
> No Android ela se instala como **"Price Local"** (`com.example.price.local`),
> ao lado do app de produção, sem substituí-lo. Os IDs de produto dos
> servidores locais são diferentes dos de produção: as listas de um app não
> comparam no outro.

**Price** é um aplicativo mobile desenvolvido em Flutter para comparação de preços de supermercados em tempo real, permitindo que os usuários economizem nas suas compras do dia a dia comparando o valor total de sua lista de compras em diferentes estabelecimentos.

## 📱 Recursos Principais

- **Pesquisa e Filtros de Produtos**: Busca inteligente por nome e categorias (como Mercearia, Hortifruti) com ordenações personalizadas (menor preço, etc.).
- **Lista de compras salva no aparelho**: itens e quantidades continuam lá quando o app é reaberto.
- **Comparador de listas**: o total da lista em cada mercado, o que falta em cada um, o item mais barato em cada mercado e a melhor combinação de mercados — veja [Lista de compras e comparação](#-lista-de-compras-e-comparação).
- **Distância até os mercados**: km reais a partir do GPS do aparelho. Na tela de busca, tocar na localização no topo lê o GPS de novo e atualiza distâncias e comparação (se o GPS estiver desligado ou a permissão bloqueada, o app explica e leva às configurações).
- **Conta e listas salvas** (aba *Perfil*): cadastro e login (`price-auth`); com a conta, a lista de compras montada pode ser salva com um nome, guardada na conta (`price-user`) e trazida de volta depois com um toque. O app segue funcionando sem conta.
- **Onde você está**: na aba *Perfil*, o botão *Atualizar minha localização* lê o GPS de novo, mostra o endereço (rua, bairro e cidade) e refaz distâncias e comparação.
- **Sem GPS**: o link *Ver todos os mercados do meu município*, na abertura, segue sem ler a posição do aparelho — o `price-back` usa um ponto de referência em Teresina e devolve os mercados num raio de 5 km dele.
- **Abas**: Busca, Lista, Comparar e Perfil. A aba Economia está comentada em `lib/modules/dashboard/dashboard_page.dart` por enquanto.
- **Interface Moderna**: Design limpo utilizando Material Design 3, identidade visual azul (`#2A5CB9`) e fonte Alexandria em transições suaves.

## 🛠️ Tecnologias Utilizadas

- **Flutter & Dart** (SDK ^3.9.2)
- **Gerenciamento de Estado**: `provider` (gerenciamento simples e reativo)
- **Layout & Estilo**: Material Design 3 com tema customizado, `google_fonts` (Alexandria) e `flutter_svg` para a logo
- **Backend**: consome a API de produção (`https://api.pricebrasil.com`) via `http` — ou os serviços locais, ver [Backend](#-backend)

## 📁 Estrutura do Projeto

```text
lib/
├── core/
│   ├── components/      # Componentes e Widgets compartilhados (ex: botões, cards, logo)
│   ├── models/          # Modelos de dados (Product, CartItem, Market, etc.)
│   ├── services/        # ApiClient/ApiService (integração com price-back), AppState, lista salva e localização
│   └── theme/           # Definições de cores e tipografia (AppTheme)
├── modules/
│   ├── cart/            # Tela do carrinho de compras
│   ├── comparison/      # Tela de comparação de preços entre supermercados
│   ├── dashboard/       # Tela principal do app (Home)
│   ├── economy/         # Tela de economia acumulada (aba comentada por enquanto)
│   ├── markets/         # Lista de mercados cadastrados
│   ├── search/          # Tela de busca de produtos
│   └── user/            # Perfil: login/cadastro, localização atual e listas salvas
├── main.dart            # Ponto de entrada do aplicativo
└── splash.dart          # Tela de abertura / splash screen
```

## 🧺 Lista de compras e comparação

O app monta uma lista de itens do **catálogo** (o mesmo produto, seja qual for o
mercado que o vende) e pergunta ao `price-back` (`POST /api/comparisons/list`)
quanto ela custa em cada mercado. Nada é calculado no aparelho: os totais, o que
falta e o "mais barato" vêm do servidor — a regra completa está no README do
`price-back`, seção "Comparação de listas".

**Busca.** A API devolve uma linha por oferta de cada mercado; a busca agrupa as
ofertas do mesmo produto numa linha só (a mais barata), com "em N mercados", o
tamanho da embalagem e o preço por kg/L (`core/services/listing_grouping.dart`).

**Tela de comparação** (`modules/comparison`):

| O que aparece | Como ler |
|---|---|
| **Melhor preço** | o mercado mais barato *que tem a lista inteira*. A diferença para o mais caro só considera mercados completos |
| Mercado **parcial** | falta item ("faltam 1 de 2 itens"). O total não é comparável: um mercado com metade da lista sai barato só porque falta coisa. Nunca se inventa preço para o item que ele não vende |
| **Aceitar outras marcas** | o mercado sem a marca da sua lista completa com outra marca do mesmo produto — só para arroz, feijão, açúcar e similares, nunca cerveja ou vinho. A linha aparece como "outra marca: Sofia" |
| Check azul numa linha | nenhum mercado vende aquele item mais barato |
| **Comprando em N mercados** | cada item onde é mais barato. O cálculo **não considera o deslocamento**, então o app mostra as paradas junto com a economia (às vezes centavos) para a pessoa decidir |
| "Nenhum mercado tem…" | itens que ninguém vende |

Trocar de marca vem **ligado por padrão**, com o interruptor no topo da tela.
Só a lista é guardada no aparelho (`SharedPreferences`, chave `shopping_list_v1`);
os preços são sempre buscados de novo.

Só produtos que estão no catálogo entram na comparação. Se a lista tiver um item
fora dele, a tela explica que ele precisa ser removido e adicionado de novo pela
busca.

### Testes

```bash
flutter test
```

Os testes do comparador usam um back simulado (`test/comparison_test_helpers.dart`)
que devolve o mesmo formato da API real, e verificam a tela em 360 e 411 px de largura.

## 🔌 Backend

Por padrão o app usa a **API de produção**, `https://api.pricebrasil.com`,
que atende todas as rotas num endereço só. Para apontar para outro endereço:
`--dart-define=API_BASE_URL=<url>`.

Para desenvolver com os serviços locais, passe o host deles com
`--dart-define=API_HOST=<ip>`: o IP de rede do computador num celular no mesmo
Wi-Fi, `10.0.2.2` no emulador Android ou `localhost` no resto. Aí o app fala
com os três serviços Go, cada um em sua porta (`ApiPorts` em
`lib/core/services/api_service.dart`):

| Serviço | Porta | O app usa para |
|---|---|---|
| `price-back` | `8080` | mercados, produtos, comparação de listas e o endereço da posição atual |
| `price-auth` | `8081` | criar conta (`/auth/signup`) e entrar (`/auth/login`) |
| `price-user` | `8082` | listas salvas na conta e economia |

Suba o Postgres em Docker e os três serviços antes — veja o README de cada um.
O `JWT_SECRET` do `price-user` precisa ser igual ao do `price-auth`. O painel
de empresas parceiras não faz parte deste app: fica nas páginas web
`price-admin-web` e `price-empresa-web`.

Os IDs de produto e de catálogo de produção e dos serviços locais são
diferentes: uma lista montada num ambiente não compara no outro, e o app pede
para limpar a lista.

A sessão (token) fica guardada no aparelho com `shared_preferences`.

## 🚀 Como Iniciar

### Pré-requisitos

Certifique-se de ter o SDK do Flutter configurado em seu ambiente de desenvolvimento.

```bash
flutter doctor
```

### Configuração e Execução

1. Baixe as dependências do projeto:
   ```bash
   flutter pub get
   ```

2. Execute o aplicativo no seu emulador ou dispositivo conectado:
   ```bash
   flutter run
   ```
