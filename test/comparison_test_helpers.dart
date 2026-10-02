import 'dart:convert';

import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:price/core/models/models.dart';
import 'package:price/core/services/api_client.dart';
import 'package:price/core/services/api_service.dart';
import 'package:price/core/services/app_state.dart';
import 'package:price/core/services/cart_storage.dart';
import 'package:price/core/services/location_service.dart';
import 'package:price/core/services/settings_storage.dart';

Product product(
  String id, {
  String? catalogId,
  String name = 'Produto',
  double price = 10,
  String market = 'Atacadão',
}) {
  return Product(
    id: id,
    name: name,
    unit: 'un',
    category: 'OUTROS',
    defaultMarketName: market,
    defaultPrice: price,
    catalogId: catalogId,
  );
}

Map<String, dynamic> marketJson(String id, String name) => {
      'id': id,
      'name': name,
      'initials': name.substring(0, 2).toUpperCase(),
      'primary_color': '#0C63E4',
      'background_color': '#E5F1FC',
    };

Map<String, dynamic> lineJson(
  String catalogId,
  String name, {
  double price = 10,
  int quantity = 1,
  bool cheapest = false,
  bool substitute = false,
  String brand = '',
}) =>
    {
      'catalog_id': catalogId,
      'name': name,
      'quantity': quantity,
      'product_id': 'p-$catalogId',
      'product_name': name,
      'brand': brand,
      'price': price,
      'line_total': price * quantity,
      'unit_price': null,
      'is_substitute': substitute,
      'is_cheapest': cheapest,
    };

/// A comparison of a 2-item list where market A has everything and market B
/// lacks the rice — the shape POST /api/comparisons/list returns.
Map<String, dynamic> comparisonJson({bool allowEquivalents = false}) {
  final a = marketJson('a', 'Atacadão - Primavera');
  final b = marketJson('b', 'Carvalho Super');
  // Round-trip through JSON so every map is Map<String, dynamic> and tests can
  // freely overwrite values (even with null).
  return jsonDecode(jsonEncode(_comparison(allowEquivalents, a, b))) as Map<String, dynamic>;
}

Map<String, dynamic> _comparison(bool allowEquivalents, Map<String, dynamic> a, Map<String, dynamic> b) {
  return {
    'allow_equivalents': allowEquivalents,
    'items': [
      {'catalog_id': 'cafe', 'name': 'Café Melitta 500g', 'quantity': 1, 'available': true, 'cheapest_market_id': 'b', 'cheapest_price': 25.9},
      {'catalog_id': 'arroz', 'name': 'Arroz Camil 5kg', 'quantity': 1, 'available': true, 'cheapest_market_id': 'a', 'cheapest_price': 22.69},
    ],
    'markets': [
      {
        'market': a,
        'distance_km': 3.96,
        'total_cost': 51.59,
        'is_complete': true,
        'found_items': 2,
        'total_items': 2,
        'substitutes_count': 0,
        'difference': 0,
        'lines': [
          lineJson('cafe', 'Café Melitta 500g', price: 28.9),
          lineJson('arroz', 'Arroz Camil 5kg', price: 22.69, cheapest: true),
        ],
        'missing': [],
      },
      {
        'market': b,
        'distance_km': 0.24,
        'total_cost': 25.9,
        'is_complete': false,
        'found_items': 1,
        'total_items': 2,
        'substitutes_count': 0,
        'difference': 0,
        'lines': [lineJson('cafe', 'Café Melitta 500g', price: 25.9, cheapest: true)],
        'missing': [
          {'catalog_id': 'arroz', 'name': 'Arroz Camil 5kg', 'quantity': 1},
        ],
      },
    ],
    'best_single_market_id': 'a',
    'best_combination': {
      'total_cost': 48.59,
      'savings_vs_best_single': 3.0,
      'stops': [
        {
          'market': a,
          'distance_km': 3.96,
          'subtotal': 22.69,
          'lines': [lineJson('arroz', 'Arroz Camil 5kg', price: 22.69, cheapest: true)],
        },
        {
          'market': b,
          'distance_km': 0.24,
          'subtotal': 25.9,
          'lines': [lineJson('cafe', 'Café Melitta 500g', price: 25.9, cheapest: true)],
        },
      ],
    },
    'max_savings': 0,
    'max_savings_percent': 0,
  };
}

/// Every request the fake backend received, decoded.
class FakeBackend {
  final List<Map<String, dynamic>> compareRequests = [];

  /// Every URL requested, in order (markets, products, comparisons...).
  final List<Uri> urls = [];

  /// What POST /api/comparisons/list answers; can be swapped per test.
  Map<String, dynamic> Function(Map<String, dynamic> request) onCompare =
      (request) => comparisonJson(allowEquivalents: request['allow_equivalents'] == true);

  /// Status code of the comparison answer (anything but 200 sends an error body).
  int compareStatus = 200;

  /// How long to hold the n-th (0-based) comparison request before answering.
  Duration Function(int index) delayFor = (_) => Duration.zero;

  http.Client get client => MockClient((request) async {
        urls.add(request.url);
        if (request.url.path == '/api/comparisons/list') {
          final body = jsonDecode(request.body) as Map<String, dynamic>;
          final index = compareRequests.length;
          compareRequests.add(body);
          await Future<void>.delayed(delayFor(index));
          if (compareStatus != 200) {
            return http.Response(
              jsonEncode({'error': 'product is not in the catalog yet'}),
              compareStatus,
              headers: {'content-type': 'application/json; charset=utf-8'},
            );
          }
          return http.Response(
            jsonEncode(onCompare(body)),
            200,
            headers: {'content-type': 'application/json; charset=utf-8'},
          );
        }
        return http.Response('[]', 200, headers: {'content-type': 'application/json'});
      });
}

AppState buildState(
  FakeBackend backend, {
  CartStorage? storage,
  LocationService? location,
  SettingsStorage? settings,
}) {
  final api = ApiClient(baseUrl: 'http://test', client: backend.client);
  return AppState(
    api: ApiService(productsClient: api, userClient: api),
    location: location,
    storage: storage ?? MemoryCartStorage(),
    settings: settings ?? MemorySettingsStorage(),
  );
}
