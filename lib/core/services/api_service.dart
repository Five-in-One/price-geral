import '../models/account.dart';
import '../models/list_comparison.dart';
import '../models/models.dart';
import 'api_client.dart';

/// Ports each local backend service listens on (see each service's README).
class ApiPorts {
  static const int products = 8080; // price-back: markets, products, comparisons
  static const int auth = 8081; // price-auth: register, login, me
  static const int user = 8082; // price-user: savings history, economy summary
}

/// Typed wrapper around every backend route the app calls. In production one
/// API serves them all; locally they are spread across the three independent
/// services (price-back, price-auth, price-user).
class ApiService {
  final ApiClient _productsClient;
  final ApiClient _authClient;
  final ApiClient _userClient;

  ApiService({ApiClient? productsClient, ApiClient? authClient, ApiClient? userClient})
    : _productsClient = productsClient ?? _defaultClient(ApiPorts.products),
      _authClient = authClient ?? _defaultClient(ApiPorts.auth),
      _userClient = userClient ?? _defaultClient(ApiPorts.user);

  static ApiClient? _production;

  static ApiClient _defaultClient(int localPort) {
    if (ApiClient.usesLocalServices) return ApiClient(baseUrl: ApiClient.hostUrlForPort(localPort));
    return _production ??= ApiClient(baseUrl: ApiClient.productionUrl);
  }

  /// Products per request; production caps `limit` at 200 and answers only
  /// 50 without it.
  static const _productsPageSize = 200;

  /// Stops paging after this many pages, whatever the server answers.
  static const _maxProductPages = 50;

  // --- price-back: markets, products, comparisons ---------------------

  Future<List<Market>> fetchMarkets({double? lat, double? lng, double? radiusKm}) async {
    final query = <String, String>{
      if (lat != null) 'lat': lat.toString(),
      if (lng != null) 'lng': lng.toString(),
      if (radiusKm != null) 'radius_km': radiusKm.toString(),
    };
    final data = await _productsClient.get('/api/markets', query: query);
    return (data as List<dynamic>).map((e) => Market.fromJson(e as Map<String, dynamic>)).toList();
  }

  Future<List<Product>> fetchProducts({String? category, String? search, bool sortByPriceAsc = false}) async {
    final query = <String, String>{
      if (category != null && category.isNotEmpty) 'category': category,
      if (search != null && search.isNotEmpty) 'search': search,
      if (sortByPriceAsc) 'sort': 'price_asc',
      'limit': '$_productsPageSize',
    };
    // Production pages the results; the local price-back ignores limit/offset
    // and answers everything at once, which ends the loop after one request.
    final products = <Product>[];
    final seen = <String>{};
    for (var page = 0; page < _maxProductPages; page++) {
      final data = await _productsClient.get(
        '/api/products',
        query: {...query, 'offset': '${page * _productsPageSize}'},
      );
      final batch = (data as List<dynamic>).map((e) => Product.fromJson(e as Map<String, dynamic>)).toList();
      final fresh = batch.where((p) => seen.add(p.id)).toList();
      products.addAll(fresh);
      if (batch.length != _productsPageSize || fresh.isEmpty) break;
    }
    return products;
  }

  /// Prices the shopping list at every market (POST /api/comparisons/list).
  /// Each item is sent as its catalog item, so the same product is found at
  /// every market that sells it; a product with no catalog item falls back to
  /// its product id (the server then says it can't be compared).
  Future<ListComparison> compareList(
    List<CartItem> items, {
    bool allowEquivalents = false,
    double? lat,
    double? lng,
    double? radiusKm,
  }) async {
    final body = {
      'items': [
        for (final item in items)
          {
            if (item.product.catalogId != null) 'catalog_id': item.product.catalogId else 'product_id': item.product.id,
            'quantity': item.quantity,
          },
      ],
      'allow_equivalents': allowEquivalents,
      if (lat != null && lng != null) ...{'lat': lat, 'lng': lng, if (radiusKm != null) 'radius_km': radiusKm},
    };
    try {
      final data = await _productsClient.post('/api/comparisons/list', body: body);
      return ListComparison.fromJson(data as Map<String, dynamic>);
    } on ApiException catch (e) {
      // The server's messages are for developers; say what the person can do.
      // Production sends a code; the local price-back only a message.
      if (e.code == 'product_not_in_catalog' || e.message.contains('not in the catalog')) {
        throw ApiException(
          'Algum item da sua lista ainda não está no catálogo e não pode ser comparado entre mercados. '
          'Remova os itens e adicione de novo pela busca.',
          statusCode: e.statusCode,
          code: e.code,
        );
      }
      if (e.code == 'unknown_product' ||
          e.code == 'unknown_catalog_item' ||
          e.message.contains('unknown product_id') ||
          e.message.contains('unknown catalog_id')) {
        throw ApiException(
          'Algum item da sua lista não existe mais. Limpe a lista e monte de novo.',
          statusCode: e.statusCode,
          code: e.code,
        );
      }
      rethrow;
    }
  }

  /// The address at [lat],[lng] (GET /api/geocode/reverse). Throws
  /// [ApiException] when the position has no known address or the lookup is down.
  Future<PlaceAddress> reverseGeocode(double lat, double lng) async {
    final data = await _productsClient.get(
      '/api/geocode/reverse',
      query: {'lat': lat.toString(), 'lng': lng.toString()},
    );
    return PlaceAddress.fromJson(data as Map<String, dynamic>);
  }

  // --- price-auth: sign up, log in -------------------------------------

  /// Creates an app account and signs it in (POST /auth/signup).
  Future<AuthSession> signUp(String username, String password) => _authenticate('/auth/signup', username, password);

  /// Signs in with an existing account (POST /auth/login).
  Future<AuthSession> logIn(String username, String password) => _authenticate('/auth/login', username, password);

  Future<AuthSession> _authenticate(String path, String username, String password) async {
    try {
      final data = await _authClient.post(path, body: {'username': username, 'password': password});
      return AuthSession.fromAuthResponse(data as Map<String, dynamic>);
    } on ApiException catch (e) {
      throw _translated(e, _authMessages);
    }
  }

  static const _authMessages = {
    // Codes (production).
    'username_taken': 'Esse nome de usuário já está em uso.',
    'rate_limited': 'Muitas tentativas. Espere um pouco e tente de novo.',
    // Messages (production and local price-auth).
    'invalid username or password': 'Usuário ou senha incorretos.',
    'username already taken': 'Esse nome de usuário já está em uso.',
    'username must be 3-32 characters with no spaces': 'O usuário precisa ter de 3 a 32 caracteres, sem espaços.',
    'username must be between 3 and 32 characters': 'O usuário precisa ter de 3 a 32 caracteres, sem espaços.',
    'username must not contain spaces': 'O usuário precisa ter de 3 a 32 caracteres, sem espaços.',
    'password must be at least 8 characters': 'A senha precisa ter pelo menos 8 caracteres.',
  };

  // --- price-user: saved lists ------------------------------------------

  Future<List<SavedList>> fetchSavedLists(String token) async {
    final data = await _userClient.get('/api/shopping-lists', token: token);
    return (data as List<dynamic>).map((e) => SavedList.fromJson(e as Map<String, dynamic>)).toList();
  }

  /// Saves [items] under [name]; a list the person already has with that name
  /// is replaced.
  Future<SavedList> saveList(String token, String name, List<CartItem> items) async {
    try {
      final data = await _userClient.post(
        '/api/shopping-lists',
        token: token,
        body: {
          'name': name,
          'items': [for (final item in items) item.toJson()],
        },
      );
      return SavedList.fromJson(data as Map<String, dynamic>);
    } on ApiException catch (e) {
      throw _translated(e, _savedListMessages);
    }
  }

  Future<void> deleteSavedList(String token, String id) async {
    await _userClient.delete('/api/shopping-lists/$id', token: token);
  }

  static const _savedListMessages = {
    // Codes (production).
    'list_limit_reached': 'Você já tem 50 listas salvas. Apague uma para salvar outra.',
    'payload_too_large': 'A lista é grande demais para salvar. Remova alguns itens.',
    // Messages (production and local price-user).
    'name must be 1-60 characters': 'O nome da lista precisa ter de 1 a 60 caracteres.',
    'name must be between 1 and 60 characters': 'O nome da lista precisa ter de 1 a 60 caracteres.',
    'list must have 1-200 items': 'A lista precisa ter entre 1 e 200 itens.',
    'items must have between 1 and 200 items': 'A lista precisa ter entre 1 e 200 itens.',
    'you already have 50 saved lists; delete one first': 'Você já tem 50 listas salvas. Apague uma para salvar outra.',
  };

  /// The server's messages are for developers; swap the ones we know (by
  /// code first, then by message) for something the person can act on,
  /// keeping the status and code for the caller.
  ApiException _translated(ApiException e, Map<String, String> messages) {
    final message = messages[e.code] ?? messages[e.message];
    return message == null ? e : ApiException(message, statusCode: e.statusCode, code: e.code);
  }

  // --- price-user: savings history, economy summary --------------------
  // Both belong to the signed-in person, so they need their token.

  Future<List<SavingsHistoryItem>> fetchSavingsHistory(String token) async {
    final data = await _userClient.get('/api/savings-history', token: token);
    return (data as List<dynamic>).map((e) => SavingsHistoryItem.fromJson(e as Map<String, dynamic>)).toList();
  }

  Future<EconomySummary> fetchEconomySummary(String token) async {
    final data = await _userClient.get('/api/economy/summary', token: token);
    return EconomySummary.fromJson(data as Map<String, dynamic>);
  }
}
