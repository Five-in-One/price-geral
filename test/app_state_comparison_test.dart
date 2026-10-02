import 'package:flutter_test/flutter_test.dart';
import 'package:price/core/models/models.dart';
import 'package:price/core/services/cart_storage.dart';
import 'package:price/core/services/location_service.dart';

import 'comparison_test_helpers.dart';

class _FixedLocation extends LocationService {
  const _FixedLocation();

  @override
  Future<UserLocation> currentLocation() async => const UserLocation(-5.07, -42.785);
}

void main() {
  test('the list starts empty: there is no demo cart any more', () async {
    final backend = FakeBackend();
    final state = buildState(backend);

    await state.restoreCart();

    expect(state.cartItems, isEmpty);
    expect(state.comparison, isNull);
    expect(backend.compareRequests, isEmpty, reason: 'nothing to compare, nothing to ask');
  });

  test('items are sent as catalog items; a product outside the catalog falls back to its id', () async {
    final backend = FakeBackend();
    final state = buildState(backend);

    state.updateQuantity(product('p1', catalogId: 'cafe'), 2);
    await pumpEventQueue();
    state.updateQuantity(product('p2'), 1); // not in the catalog
    await pumpEventQueue();

    final items = (backend.compareRequests.last['items'] as List).cast<Map<String, dynamic>>();
    expect(items[0], {'catalog_id': 'cafe', 'quantity': 2});
    expect(items[1], {'product_id': 'p2', 'quantity': 1});
  });

  test('accepting other brands is on by default and can be switched off', () async {
    final backend = FakeBackend();
    final state = buildState(backend);
    state.updateQuantity(product('p1', catalogId: 'cafe'), 1);
    await pumpEventQueue();

    expect(state.allowEquivalents, isTrue);
    expect(backend.compareRequests.last['allow_equivalents'], isTrue);

    state.setAllowEquivalents(false);
    await pumpEventQueue();

    expect(backend.compareRequests.last['allow_equivalents'], isFalse);
    expect(state.comparison!.allowEquivalents, isFalse);
  });

  test('sends the user position once it is known', () async {
    final backend = FakeBackend();
    final state = buildState(backend, location: const _FixedLocation());

    await state.detectLocation();
    state.updateQuantity(product('p1', catalogId: 'cafe'), 1);
    await pumpEventQueue();

    expect(backend.compareRequests.last['lat'], -5.07);
    expect(backend.compareRequests.last['lng'], -42.785);
  });

  test('without a position no lat/lng is sent', () async {
    final backend = FakeBackend();
    final state = buildState(backend);
    state.updateQuantity(product('p1', catalogId: 'cafe'), 1);
    await pumpEventQueue();

    expect(backend.compareRequests.last.containsKey('lat'), isFalse);
  });

  test('the list is saved on the device and comes back in the next session', () async {
    final storage = MemoryCartStorage();
    final first = buildState(FakeBackend(), storage: storage);
    first.updateQuantity(product('p1', catalogId: 'cafe', name: 'Café Melitta 500g'), 3);
    await pumpEventQueue();

    expect(storage.saved, hasLength(1));

    final backend = FakeBackend();
    final second = buildState(backend, storage: storage);
    await second.restoreCart();
    await pumpEventQueue();

    expect(second.cartItems.single.quantity, 3);
    expect(second.cartItems.single.product.catalogId, 'cafe');
    expect(backend.compareRequests, hasLength(1), reason: 'the restored list is compared right away');
    expect(second.comparison, isNotNull);
  });

  test('restoring never overwrites what the user already added', () async {
    final storage = MemoryCartStorage([
      CartItem(product: product('old', catalogId: 'old'), quantity: 1),
    ]);
    final state = buildState(FakeBackend(), storage: storage);

    state.updateQuantity(product('new', catalogId: 'new'), 1);
    await state.restoreCart();

    expect(state.cartItems.map((i) => i.product.id), ['new']);
  });

  test('clearing the list clears the comparison', () async {
    final storage = MemoryCartStorage();
    final state = buildState(FakeBackend(), storage: storage);
    state.updateQuantity(product('p1', catalogId: 'cafe'), 1);
    await pumpEventQueue();
    expect(state.comparison, isNotNull);

    state.clearCart();
    await pumpEventQueue();

    expect(state.comparison, isNull);
    expect(storage.saved, isEmpty);
  });

  test('a slow, older answer never overwrites a newer one', () async {
    final backend = FakeBackend();
    // the first request is held back so the second one overtakes it
    backend.delayFor = (i) => i == 0 ? const Duration(milliseconds: 80) : Duration.zero;
    final state = buildState(backend);

    state.updateQuantity(product('p1', catalogId: 'cafe'), 1); // request 0: slow, equivalents on
    await pumpEventQueue(times: 2);
    state.setAllowEquivalents(false); // request 1: fast, equivalents off

    await Future<void>.delayed(const Duration(milliseconds: 200));
    await pumpEventQueue();

    expect(backend.compareRequests, hasLength(2));
    expect(state.comparison!.allowEquivalents, isFalse,
        reason: 'the answer to the latest request wins, even though it arrived first');
    expect(state.isLoadingComparison, isFalse);
  });

  test('a server error is reported instead of leaving the screen loading', () async {
    final backend = FakeBackend()..compareStatus = 400;
    final state = buildState(backend);

    state.updateQuantity(product('p1'), 1);
    await pumpEventQueue();

    expect(state.comparison, isNull);
    expect(state.comparisonError, contains('catálogo'));
    expect(state.isLoadingComparison, isFalse);
  });
}
