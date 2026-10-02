import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:price/core/services/app_state.dart';
import 'package:price/modules/comparison/comparison_page.dart';
import 'package:provider/provider.dart';

import 'comparison_test_helpers.dart';

/// Renders the comparison tab for a 2-item list against the fake backend.
Future<AppState> pumpComparison(
  WidgetTester tester,
  FakeBackend backend, {
  double width = 411,
}) async {
  // tall enough that the whole list is built (ListView only builds what is visible)
  tester.view.physicalSize = Size(width, 2400);
  tester.view.devicePixelRatio = 1.0;
  addTearDown(tester.view.reset);

  final state = buildState(backend);
  state.updateQuantity(product('p1', catalogId: 'cafe', name: 'Café Melitta 500g'), 1);
  state.updateQuantity(product('p2', catalogId: 'arroz', name: 'Arroz Camil 5kg'), 1);

  await tester.pumpWidget(
    ChangeNotifierProvider.value(
      value: state,
      child: const MaterialApp(home: Scaffold(body: ComparisonPage())),
    ),
  );
  await tester.pumpAndSettle();
  return state;
}

void main() {
  testWidgets('shows the address of every market in the comparison, not only the best one', (tester) async {
    final backend = FakeBackend();
    backend.onCompare = (request) {
      final json = comparisonJson(allowEquivalents: request['allow_equivalents'] == true);
      final addresses = {
        'a': {'street': 'Avenida Duque de Caxias', 'number': '2732', 'neighborhood': 'Real Copagri'},
        'b': {'street': 'Avenida Homero Castelo Branco', 'number': '1819', 'neighborhood': 'Ininga'},
      };
      for (final m in json['markets'] as List) {
        (m['market'] as Map<String, dynamic>).addAll(addresses[m['market']['id']]!);
      }
      return json;
    };
    await pumpComparison(tester, backend);

    expect(tester.takeException(), isNull);
    // the best market: in the headline card and in its row
    expect(find.text('Avenida Duque de Caxias, 2732 · Real Copagri'), findsNWidgets(2));
    // the other market, without expanding it
    expect(find.text('Avenida Homero Castelo Branco, 1819 · Ininga'), findsOneWidget);
  });

  testWidgets('shows the cheapest market that has the whole list, and what the others lack', (tester) async {
    final backend = FakeBackend();
    await pumpComparison(tester, backend);

    expect(tester.takeException(), isNull);

    // headline: the complete market
    expect(find.text('MELHOR PREÇO'), findsOneWidget);
    expect(find.textContaining('R\$ 51,59 no total'), findsOneWidget);

    // the incomplete market is flagged, not presented as a bargain
    expect(find.text('faltam 1 de 2 itens'), findsOneWidget);
    expect(find.text('parcial'), findsOneWidget);
    expect(find.text('R\$ 25,90'), findsOneWidget, reason: 'its partial total is still shown, as partial');

    // only one market has everything: no invented "savings"
    expect(find.text('É o único mercado com todos os itens da lista.'), findsOneWidget);
  });

  testWidgets('never invents a price for an item a market does not sell', (tester) async {
    final backend = FakeBackend();
    await pumpComparison(tester, backend);

    await tester.tap(find.text('Carvalho Super').last);
    await tester.pumpAndSettle();

    expect(find.text('Não tem: Arroz Camil 5kg'), findsOneWidget);
    // Atacadão's rice price (R$ 22,69) must not show up here as a stand-in: with
    // only Carvalho expanded there is no place on screen where it belongs
    expect(find.text('R\$ 22,69'), findsNothing);
  });

  testWidgets('marks the line each item is cheapest at', (tester) async {
    final backend = FakeBackend();
    await pumpComparison(tester, backend);

    await tester.tap(find.text('Atacadão - Primavera').last);
    await tester.pumpAndSettle();

    // rice is cheapest at Atacadão (check icon next to its price), coffee is not
    expect(find.byIcon(Icons.check_circle_outline), findsOneWidget);
  });

  testWidgets('offers buying each item where it is cheapest, with the trade-off stated', (tester) async {
    final backend = FakeBackend();
    await pumpComparison(tester, backend);

    expect(find.text('Comprando em 2 mercados: R\$ 48,59'), findsOneWidget);
    expect(find.textContaining('R\$ 3,00 a menos do que comprar tudo em Atacadão - Primavera'), findsOneWidget);

    await tester.tap(find.textContaining('Comprando em 2 mercados'));
    await tester.pumpAndSettle();

    expect(find.text('O cálculo não considera o custo do deslocamento entre os mercados.'), findsOneWidget);
  });

  testWidgets('the "other brands" switch asks the server again', (tester) async {
    final backend = FakeBackend();
    final state = await pumpComparison(tester, backend);
    expect(state.allowEquivalents, isTrue);
    final before = backend.compareRequests.length;

    await tester.tap(find.byType(Switch));
    await tester.pumpAndSettle();

    expect(state.allowEquivalents, isFalse);
    expect(backend.compareRequests.length, greaterThan(before));
    expect(backend.compareRequests.last['allow_equivalents'], isFalse);
  });

  testWidgets('shows which brand replaced the one on the list', (tester) async {
    final backend = FakeBackend();
    backend.onCompare = (request) {
      final json = comparisonJson(allowEquivalents: true);
      final markets = (json['markets'] as List).cast<Map<String, dynamic>>();
      // Carvalho now completes the list with another brand of rice
      markets[1]
        ..['is_complete'] = true
        ..['found_items'] = 2
        ..['substitutes_count'] = 1
        ..['missing'] = []
        ..['lines'] = [
          lineJson('cafe', 'Café Melitta 500g', price: 25.9, cheapest: true),
          lineJson('arroz', 'Arroz Sofia 5kg', price: 21.9, cheapest: true, substitute: true, brand: 'Sofia'),
        ]
        ..['total_cost'] = 47.8;
      markets[0]['difference'] = 3.79;
      json['best_single_market_id'] = 'b';
      json['markets'] = [markets[1], markets[0]];
      return json;
    };
    await pumpComparison(tester, backend);

    expect(find.text('1 outra marca'), findsOneWidget);

    await tester.tap(find.text('Carvalho Super').last);
    await tester.pumpAndSettle();

    expect(find.text('outra marca: Sofia'), findsOneWidget);
  });

  testWidgets('says so when no market has every item', (tester) async {
    final backend = FakeBackend();
    backend.onCompare = (request) {
      final json = comparisonJson();
      final markets = (json['markets'] as List).cast<Map<String, dynamic>>();
      markets[0]
        ..['is_complete'] = false
        ..['found_items'] = 1;
      json['best_single_market_id'] = '';
      return json;
    };
    await pumpComparison(tester, backend);

    expect(find.text('NENHUM MERCADO TEM TODOS OS ITENS'), findsOneWidget);
    expect(find.text('MELHOR PREÇO'), findsNothing);
  });

  testWidgets('lists the items no market sells', (tester) async {
    final backend = FakeBackend();
    backend.onCompare = (request) {
      final json = comparisonJson();
      (json['items'] as List).cast<Map<String, dynamic>>()[1]
        ..['available'] = false
        ..['cheapest_market_id'] = ''
        ..['cheapest_price'] = null;
      return json;
    };
    await pumpComparison(tester, backend);

    expect(find.text('Nenhum mercado tem este item:'), findsOneWidget);
    expect(find.text('• Arroz Camil 5kg'), findsOneWidget);
  });

  testWidgets('empty list points back to the search', (tester) async {
    tester.view.physicalSize = const Size(411, 900);
    tester.view.devicePixelRatio = 1.0;
    addTearDown(tester.view.reset);

    final state = buildState(FakeBackend());
    await tester.pumpWidget(
      ChangeNotifierProvider.value(
        value: state,
        child: const MaterialApp(home: Scaffold(body: ComparisonPage())),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Adicione itens para ver a comparação.'), findsOneWidget);
  });

  testWidgets('a failed comparison shows the reason instead of a spinner', (tester) async {
    final backend = FakeBackend()..compareStatus = 400;
    await pumpComparison(tester, backend);

    expect(find.textContaining('não está no catálogo'), findsOneWidget);
    expect(find.byType(CircularProgressIndicator), findsNothing);
  });

  for (final width in [360.0, 411.0]) {
    testWidgets('does not overflow at ${width.toInt()}px wide, even with everything expanded', (tester) async {
      final backend = FakeBackend();
      await pumpComparison(tester, backend, width: width);

      await tester.tap(find.textContaining('Comprando em 2 mercados'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Carvalho Super').last);
      await tester.pumpAndSettle();
      await tester.tap(find.text('Atacadão - Primavera').last);
      await tester.pumpAndSettle();

      expect(tester.takeException(), isNull);
    });
  }
}
