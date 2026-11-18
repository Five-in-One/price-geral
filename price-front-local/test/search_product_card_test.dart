import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:price/core/services/api_client.dart';
import 'package:price/core/services/api_service.dart';
import 'package:price/core/services/app_state.dart';
import 'package:price/modules/search/search_page.dart';
import 'package:provider/provider.dart';

// The product from the phone screenshot where the row
// "R$ 1,29 /g • @ Carvalho Super - Homero Castelo Branco" overflowed by 26px.
const _longMarket = 'Carvalho Super - Homero Castelo Branco';

http.Client _fakeBackend() {
  return MockClient((request) async {
    if (request.url.path == '/api/products/search') {
      return http.Response(
        jsonEncode([
          {
            'id': 'p1',
            'name': 'Macarrao Instataneo Marata 74g Galinha Caipira',
            'unit': 'g',
            'category': 'GRÃOS',
            'default_market_name': _longMarket,
            'default_price': 1.29,
          },
          {
            'id': 'p2',
            'name': 'Molho de Tomate Tomador 300g',
            'unit': 'pct',
            'category': 'GRÃOS',
            'default_market_name': 'Atacadão - Primavera',
            'default_price': 1.25,
          },
        ]),
        200,
        headers: {'content-type': 'application/json; charset=utf-8'},
      );
    }
    return http.Response('[]', 200);
  });
}

Future<void> _pumpSearch(WidgetTester tester, Size size) async {
  tester.view.physicalSize = size;
  tester.view.devicePixelRatio = 1.0;
  addTearDown(tester.view.reset);

  final client = ApiClient(baseUrl: 'http://test', client: _fakeBackend());
  final appState = AppState(
    api: ApiService(productsClient: client, userClient: client),
  );

  await tester.pumpWidget(
    ChangeNotifierProvider.value(
      value: appState,
      child: const MaterialApp(home: Scaffold(body: SearchPage())),
    ),
  );
  await tester.pumpAndSettle();
}

void main() {
  // 360dp is the phone from the screenshot. Not narrower: the default test font
  // (Ahem) is far wider than the real one and would overflow unrelated widgets.
  for (final width in [360.0, 411.0]) {
    testWidgets('search product cards do not overflow at ${width.toInt()}px wide', (tester) async {
      await _pumpSearch(tester, Size(width, 800));

      expect(find.text('@ $_longMarket'), findsOneWidget);
      // A RenderFlex overflow is reported as a FlutterError during layout.
      expect(tester.takeException(), isNull);
    });
  }
}
