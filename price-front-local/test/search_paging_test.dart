import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:price/core/services/api_client.dart';
import 'package:price/core/services/api_service.dart';
import 'package:price/core/services/app_state.dart';
import 'package:price/modules/dashboard/dashboard_page.dart';
import 'package:price/modules/search/search_page.dart';
import 'package:provider/provider.dart';

/// A price-back that pages [total] products the way GET /api/products/search
/// does, and records every search it answered.
class _PagedBackend {
  final int total;
  final List<Map<String, String>> searches = [];

  _PagedBackend(this.total);

  http.Client get client => MockClient((request) async {
    final q = request.url.queryParameters;
    if (request.url.path == '/api/products/categories') {
      return _json([
        {'category': 'GRÃOS', 'count': 30},
        {'category': 'CARNES', 'count': 15},
      ]);
    }
    if (request.url.path != '/api/products/search') return _json([]);
    searches.add(q);
    final offset = int.parse(q['offset']!);
    final limit = int.parse(q['limit']!);
    final end = (offset + limit).clamp(0, total);
    return _json({
      'items': [
        for (var i = offset; i < end; i++)
          {
            'id': 'p$i',
            'name': 'Produto $i',
            'unit': 'un',
            'category': 'GRÃOS',
            'default_market_name': 'Mercado',
            'default_price': 1.0 + i,
          },
      ],
      'total': total,
      'limit': limit,
      'offset': offset,
      'has_more': end < total,
    });
  });

  static http.Response _json(Object body) =>
      http.Response(jsonEncode(body), 200, headers: {'content-type': 'application/json; charset=utf-8'});
}

Future<AppState> _pump(WidgetTester tester, _PagedBackend backend) async {
  tester.view.physicalSize = const Size(411, 900);
  tester.view.devicePixelRatio = 1.0;
  addTearDown(tester.view.reset);

  final client = ApiClient(baseUrl: 'http://test', client: backend.client);
  final state = AppState(
    api: ApiService(productsClient: client, userClient: client),
  );
  await tester.pumpWidget(
    ChangeNotifierProvider.value(
      value: state,
      child: const MaterialApp(home: Scaffold(body: SearchPage())),
    ),
  );
  await tester.pumpAndSettle();
  return state;
}

Future<void> _loadMore(WidgetTester tester) async {
  await _scrollToEnd(tester);
  await tester.tap(find.byKey(const ValueKey('load-more')));
  await tester.pumpAndSettle();
  await _scrollToEnd(tester);
}

Future<void> _scrollToEnd(WidgetTester tester) async {
  await tester.fling(find.byType(CustomScrollView), const Offset(0, -20000), 5000);
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('loads only the first page; the next ones come only from "Carregar mais"', (tester) async {
    final backend = _PagedBackend(45);
    final state = await _pump(tester, backend);

    expect(state.searchResults, hasLength(20));
    expect(find.text('20 de 45 produtos · página 1 de 3'), findsOneWidget);
    expect(backend.searches.map((q) => q['offset']), ['0']);

    // Scrolling to the end asks for nothing by itself.
    await _scrollToEnd(tester);
    expect(backend.searches.map((q) => q['offset']), ['0']);
    expect(find.text('Carregar mais 20'), findsOneWidget);

    await _loadMore(tester);
    expect(state.searchResults, hasLength(40));
    expect(find.text('Carregar mais 5'), findsOneWidget, reason: 'only 5 are left');

    await _loadMore(tester);
    expect(state.searchResults, hasLength(45));
    expect(state.searchHasMore, isFalse);
    expect(backend.searches.map((q) => q['offset']), ['0', '20', '40']);
    expect(find.byKey(const ValueKey('load-more')), findsNothing);
    expect(find.text('Fim dos resultados · 3 páginas'), findsOneWidget);
    expect((state.searchPagesLoaded, state.searchPageCount), (3, 3));
  });

  testWidgets('categories come from the server, and picking one starts over from the first page', (tester) async {
    final backend = _PagedBackend(45);
    final state = await _pump(tester, backend);
    await _loadMore(tester);
    expect(state.searchResults.length, greaterThan(20));

    await tester.drag(find.byType(CustomScrollView), const Offset(0, 20000));
    await tester.pumpAndSettle();
    await tester.ensureVisible(find.text('Carnes (15)'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Carnes (15)'));
    await tester.pumpAndSettle();

    expect(backend.searches.last['category'], 'CARNES');
    expect(backend.searches.last['offset'], '0');
    expect(state.searchResults, hasLength(20));
  });

  testWidgets('the sort and the price filter are sent to the server', (tester) async {
    final backend = _PagedBackend(5);
    final state = await _pump(tester, backend);

    await tester.tap(find.byKey(const ValueKey('search-sort')));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Maior preço'));
    await tester.pumpAndSettle();
    expect(backend.searches.last['sort'], 'price_desc');

    await tester.tap(find.byKey(const ValueKey('search-filters')));
    await tester.pumpAndSettle();
    await tester.enterText(find.byKey(const ValueKey('filter-min-price')), '2,50');
    await tester.tap(find.byKey(const ValueKey('filter-apply')));
    await tester.pumpAndSettle();

    expect(backend.searches.last['min_price'], '2.50');
    expect(backend.searches.last['sort'], 'price_desc');
    expect(state.searchFilters.activeCount, 1);
    expect(find.text('Filtros (1)'), findsOneWidget);
  });

  scrollToTopTests();
}

double _toTopOpacity(WidgetTester tester) => tester
    .widget<AnimatedOpacity>(
      find.ancestor(of: find.byKey(const ValueKey('scroll-to-top')), matching: find.byType(AnimatedOpacity)),
    )
    .opacity;

void scrollToTopTests() {
  testWidgets('the "Topo" button shows while scrolling up from far down, and takes the list to the top', (
    tester,
  ) async {
    final backend = _PagedBackend(45);
    await _pump(tester, backend);
    final scrollable = find.byType(CustomScrollView);
    double offset() => tester.state<ScrollableState>(find.byType(Scrollable).first).position.pixels;

    expect(_toTopOpacity(tester), 0, reason: 'hidden at the top');

    await tester.drag(scrollable, const Offset(0, -2500));
    await tester.pumpAndSettle();
    expect(_toTopOpacity(tester), 0, reason: 'hidden while going down');

    await tester.drag(scrollable, const Offset(0, 300));
    await tester.pumpAndSettle();
    expect(offset(), greaterThan(SearchPage.scrollToTopThreshold));
    expect(_toTopOpacity(tester), 1, reason: 'shown when going back up');

    await tester.tap(find.byKey(const ValueKey('scroll-to-top')));
    await tester.pumpAndSettle();
    expect(offset(), 0);
    expect(_toTopOpacity(tester), 0);
  });

  testWidgets('the "Topo" button stays after an upward fling, until the list goes down again', (tester) async {
    await _pump(tester, _PagedBackend(45));
    final scrollable = find.byType(CustomScrollView);

    await tester.drag(scrollable, const Offset(0, -3000));
    await tester.pumpAndSettle();
    await tester.fling(scrollable, const Offset(0, 250), 1500);
    await tester.pumpAndSettle();
    expect(_toTopOpacity(tester), 1);

    await tester.drag(scrollable, const Offset(0, -200));
    await tester.pumpAndSettle();
    expect(_toTopOpacity(tester), 0);
  });

  testWidgets('tapping "Busca" while on the search takes the list to the top', (tester) async {
    tester.view.physicalSize = const Size(411, 900);
    tester.view.devicePixelRatio = 1.0;
    addTearDown(tester.view.reset);
    final client = ApiClient(baseUrl: 'http://test', client: _PagedBackend(45).client);
    final state = AppState(
      api: ApiService(productsClient: client, userClient: client),
    );
    await tester.pumpWidget(
      ChangeNotifierProvider.value(
        value: state,
        child: const MaterialApp(home: DashboardPage()),
      ),
    );
    await tester.pumpAndSettle();
    final list = find.descendant(of: find.byType(SearchPage), matching: find.byType(Scrollable)).first;
    double offset() => tester.state<ScrollableState>(list).position.pixels;

    await tester.drag(list, const Offset(0, -2500));
    await tester.pumpAndSettle();
    expect(offset(), greaterThan(1000));

    await tester.tap(find.text('Busca').last);
    await tester.pumpAndSettle();
    expect(offset(), 0);
    expect(state.currentTabIndex, 0);
  });
}
