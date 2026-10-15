import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:price/core/services/api_client.dart';
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

  test('products are fetched page by page until a short page', () async {
    final offsets = <String?>[];
    final service = _service(
      MockClient((request) async {
        final offset = int.parse(request.url.queryParameters['offset']!);
        offsets.add(request.url.queryParameters['offset']);
        expect(request.url.queryParameters['limit'], '200');
        final count = offset == 400 ? 70 : 200; // 470 products in all
        return http.Response(
          jsonEncode([for (var i = 0; i < count; i++) _product(offset + i)]),
          200,
          headers: {'content-type': 'application/json; charset=utf-8'},
        );
      }),
    );
    final products = await service.fetchProducts();
    expect(products, hasLength(470));
    expect(offsets, ['0', '200', '400']);
  });

  test('a server that ignores limit/offset (local price-back) is asked only once', () async {
    var calls = 0;
    final service = _service(
      MockClient((_) async {
        calls++;
        return http.Response(
          jsonEncode([for (var i = 0; i < 788; i++) _product(i)]),
          200,
          headers: {'content-type': 'application/json; charset=utf-8'},
        );
      }),
    );
    final products = await service.fetchProducts();
    expect(products, hasLength(788));
    expect(calls, 1);
  });

  test('a repeated full page ends the paging without duplicates', () async {
    var calls = 0;
    final service = _service(
      MockClient((_) async {
        calls++;
        return http.Response(
          jsonEncode([for (var i = 0; i < 200; i++) _product(i)]),
          200,
          headers: {'content-type': 'application/json; charset=utf-8'},
        );
      }),
    );
    final products = await service.fetchProducts();
    expect(products, hasLength(200));
    expect(calls, 2);
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
