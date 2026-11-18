import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:price/core/services/api_client.dart';
import 'package:price/core/models/product_search.dart';
import 'package:price/core/services/api_service.dart';

Map<String, dynamic> _product(int i) => {
  'id': 'p$i',
  'name': 'Produto $i',
  'unit': 'un',
  'category': 'OUTROS',
  'default_market_name': 'Mercado',
  'default_price': 1.0 + i,
};

http.Response _error(int status, String code, String message) => http.Response(
  jsonEncode({
    'error': {'code': code, 'message': message},
  }),
  status,
  headers: {'content-type': 'application/json; charset=utf-8'},
);

ApiService _service(MockClient backend) {
  final api = ApiClient(baseUrl: 'http://test', client: backend);
  return ApiService(productsClient: api, authClient: api, userClient: api);
}

void main() {
  test('the error envelope of the production API is read (message and code)', () async {
    final api = ApiClient(
      baseUrl: 'http://test',
      client: MockClient((_) async => _error(409, 'already_decided', 'suggestion already decided')),
    );
    await expectLater(
      api.get('/x'),
      throwsA(
        isA<ApiException>()
            .having((e) => e.message, 'message', 'suggestion already decided')
            .having((e) => e.code, 'code', 'already_decided')
            .having((e) => e.statusCode, 'statusCode', 409),
      ),
    );
  });

  test('the error of the local services is still read', () async {
    final api = ApiClient(
      baseUrl: 'http://test',
      client: MockClient((_) async => http.Response(jsonEncode({'error': 'boom'}), 400)),
    );
    await expectLater(
      api.get('/x'),
      throwsA(isA<ApiException>().having((e) => e.message, 'message', 'boom').having((e) => e.code, 'code', isNull)),
    );
  });

  test('sign-up errors are translated by code', () async {
    final service = _service(MockClient((_) async => _error(409, 'username_taken', 'username is already taken')));
    await expectLater(
      service.signUp('ana', 'senhaForte123'),
      throwsA(isA<ApiException>().having((e) => e.message, 'message', 'Esse nome de usuário já está em uso.')),
    );
  });

  test('comparison errors are translated by code', () async {
    final service = _service(
      MockClient(
        (_) async => _error(400, 'product_not_in_catalog', 'product_id p1 is not linked to a catalog item yet'),
      ),
    );
    await expectLater(
      service.compareList([]),
      throwsA(isA<ApiException>().having((e) => e.message, 'message', contains('ainda não está no catálogo'))),
    );
  });

  test('the search sends text, category, sort, filters and the page to the server', () async {
    late Uri asked;
    final service = _service(
      MockClient((request) async {
        asked = request.url;
        return http.Response(
          jsonEncode({
            'items': [
              {..._product(1), 'markets_count': 3},
            ],
            'total': 41,
            'limit': 20,
            'offset': 20,
            'has_more': true,
          }),
          200,
          headers: {'content-type': 'application/json; charset=utf-8'},
        );
      }),
    );

    final page = await service.searchProducts(
      search: ' arroz 5kg ',
      category: 'GRÃOS',
      sort: ProductSort.unitPriceAsc,
      filters: const SearchFilters(marketIds: {'b', 'a'}, minPrice: 2, maxPrice: 30.5, nearbyOnly: true),
      lat: -5.07,
      lng: -42.8,
      radiusKm: 5,
      offset: 20,
    );

    expect(asked.path, '/api/products/search');
    expect(asked.queryParameters, {
      'search': 'arroz 5kg',
      'category': 'GRÃOS',
      'market_id': 'a,b',
      'min_price': '2.00',
      'max_price': '30.50',
      'lat': '-5.07',
      'lng': '-42.8',
      'radius_km': '5.0',
      'sort': 'unit_price_asc',
      'limit': '20',
      'offset': '20',
    });
    expect(page.total, 41);
    expect(page.hasMore, isTrue);
    expect(page.items.single.marketsCount, 3);
  });

  test('"Tudo" and the radius without "nearby only" are not sent', () async {
    late Uri asked;
    final service = _service(
      MockClient((request) async {
        asked = request.url;
        return http.Response(jsonEncode({'items': [], 'total': 0, 'has_more': false}), 200);
      }),
    );

    await service.searchProducts(category: 'Tudo', lat: -5.07, lng: -42.8, radiusKm: 5);

    expect(asked.queryParameters, {'sort': 'price_asc', 'limit': '20', 'offset': '0'});
  });

  test('categories come with their counts', () async {
    final service = _service(
      MockClient((request) async {
        expect(request.url.path, '/api/products/categories');
        return http.Response(
          jsonEncode([
            {'category': 'LATICÍNIOS', 'count': 93},
          ]),
          200,
          headers: {'content-type': 'application/json; charset=utf-8'},
        );
      }),
    );

    final categories = await service.fetchCategories();

    expect(categories.single.label, 'Laticínios');
    expect(categories.single.count, 93);
  });

  test('savings history and economy summary send the token', () async {
    final auth = <String?>[];
    final service = _service(
      MockClient((request) async {
        auth.add(request.headers['Authorization']);
        final body = request.url.path == '/api/savings-history'
            ? []
            : {
                'total_saved': 0,
                'month_label': 'Setembro de 2026',
                'month_spend': 0,
                'month_savings': 0,
                'month_savings_percent': 0,
                'monthly_chart': [],
              };
        return http.Response(jsonEncode(body), 200);
      }),
    );
    await service.fetchSavingsHistory('tok');
    await service.fetchEconomySummary('tok');
    expect(auth, ['Bearer tok', 'Bearer tok']);
  });
}
