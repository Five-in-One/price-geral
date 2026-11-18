import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:price/core/components/market_address.dart';
import 'package:price/core/models/models.dart';

Market _market(Map<String, dynamic> address) => Market.fromJson({
      'id': 'fe',
      'name': 'FerreiraEmCasa',
      'initials': 'FE',
      'primary_color': '#D97706',
      'background_color': '#FDF0E2',
      'distance_km': 0.83,
      ...address,
    });

const _fullAddress = {
  'cep': '64001-160',
  'street': 'Rua Félix Pacheco',
  'number': '965',
  'neighborhood': 'Centro Sul',
  'city': 'Teresina',
  'state': 'PI',
};

void main() {
  test('reads the address the API sends', () {
    final market = _market(_fullAddress);

    expect(market.hasAddress, isTrue);
    expect(market.streetLine, 'Rua Félix Pacheco, 965');
    expect(market.regionLine, 'Centro Sul, Teresina - PI');
    expect(market.fullAddress, 'Rua Félix Pacheco, 965, Centro Sul, Teresina - PI, CEP 64001-160');
  });

  test('pieces the market did not fill in are left out', () {
    final market = _market({'street': 'Rua dos Pombos', 'complement': 'Loja 2', 'city': 'Teresina'});

    expect(market.streetLine, 'Rua dos Pombos - Loja 2');
    expect(market.regionLine, 'Teresina');
    expect(market.fullAddress, 'Rua dos Pombos - Loja 2, Teresina');
  });

  test('a market without an address says so', () {
    final market = _market({});

    expect(market.hasAddress, isFalse);
    expect(market.fullAddress, '');
  });

  test('the map link searches for the market name and address', () {
    final uri = mapsUriFor(_market(_fullAddress));

    expect(uri.host, 'www.google.com');
    expect(uri.queryParameters['api'], '1');
    expect(uri.queryParameters['query'], 'FerreiraEmCasa, Rua Félix Pacheco, 965, Centro Sul, Teresina - PI, CEP 64001-160');
  });

  testWidgets('the details sheet shows the whole address and the actions', (tester) async {
    final market = _market(_fullAddress);
    await tester.pumpWidget(MaterialApp(
      home: Scaffold(
        body: Builder(
          builder: (context) => TextButton(
            onPressed: () => showMarketDetails(context, market),
            child: const Text('abrir'),
          ),
        ),
      ),
    ));

    await tester.tap(find.text('abrir'));
    await tester.pumpAndSettle();

    expect(find.text('FerreiraEmCasa'), findsOneWidget);
    expect(find.text('0,8 km de você'), findsOneWidget);
    expect(find.text('Rua Félix Pacheco, 965'), findsOneWidget);
    expect(find.text('Centro Sul, Teresina - PI'), findsOneWidget);
    expect(find.text('CEP 64001-160'), findsOneWidget);
    expect(find.text('Abrir no mapa'), findsOneWidget);
    expect(find.text('Copiar endereço'), findsOneWidget);
  });

  testWidgets('without an address the sheet says so and offers a map search', (tester) async {
    final market = _market({});
    await tester.pumpWidget(MaterialApp(
      home: Scaffold(
        body: Builder(
          builder: (context) => TextButton(
            onPressed: () => showMarketDetails(context, market),
            child: const Text('abrir'),
          ),
        ),
      ),
    ));

    await tester.tap(find.text('abrir'));
    await tester.pumpAndSettle();

    expect(find.text('Este mercado ainda não cadastrou o endereço.'), findsOneWidget);
    expect(find.text('Procurar no mapa'), findsOneWidget);
    expect(find.text('Copiar endereço'), findsNothing);
  });

  testWidgets('the address line shows street and neighborhood', (tester) async {
    await tester.pumpWidget(MaterialApp(home: Scaffold(body: MarketAddressLine(market: _market(_fullAddress)))));

    expect(find.text('Rua Félix Pacheco, 965 · Centro Sul'), findsOneWidget);
  });
}
