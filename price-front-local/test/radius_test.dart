import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:price/core/components/radius_selector.dart';
import 'package:price/core/services/app_state.dart';
import 'package:price/core/services/location_service.dart';
import 'package:price/core/services/settings_storage.dart';
import 'package:provider/provider.dart';

import 'comparison_test_helpers.dart';

class _FixedLocation extends LocationService {
  const _FixedLocation();

  @override
  Future<UserLocation> currentLocation() async => const UserLocation(-5.07, -42.785);
}

Iterable<Uri> _marketCalls(FakeBackend backend) => backend.urls.where((u) => u.path == '/api/markets');

void main() {
  test('markets are searched within the default radius of 5 km', () async {
    final backend = FakeBackend();
    final state = buildState(backend);

    await state.loadMarkets();

    expect(state.radiusKm, AppState.defaultRadiusKm);
    expect(_marketCalls(backend).last.queryParameters['radius_km'], '5.0');
  });

  test('changing the radius reloads the markets with it and saves it', () async {
    final backend = FakeBackend();
    final settings = MemorySettingsStorage();
    final state = buildState(backend, settings: settings);

    await state.setRadiusKm(20);

    expect(state.radiusKm, 20);
    expect(_marketCalls(backend).last.queryParameters['radius_km'], '20.0');
    expect(settings.radiusKm, 20);
  });

  test('the radius stays between 1 and 50 km and is rounded to whole km', () async {
    final state = buildState(FakeBackend());

    await state.setRadiusKm(0.2);
    expect(state.radiusKm, AppState.minRadiusKm);

    await state.setRadiusKm(500);
    expect(state.radiusKm, AppState.maxRadiusKm);

    await state.setRadiusKm(12.6);
    expect(state.radiusKm, 13);
  });

  test('the radius chosen in a previous session is restored', () async {
    final state = buildState(FakeBackend(), settings: MemorySettingsStorage(15));

    await state.restoreRadius();

    expect(state.radiusKm, 15);
  });

  test('the comparison uses the radius once the position is known', () async {
    final backend = FakeBackend();
    final state = buildState(backend, location: const _FixedLocation());
    state.updateQuantity(product('p1', catalogId: 'cafe'), 1);
    await pumpEventQueue();
    expect(backend.compareRequests.last.containsKey('radius_km'), isFalse,
        reason: 'without a position there is no distance to limit');

    await state.updateLocation();
    await state.setRadiusKm(8);

    expect(backend.compareRequests.last['radius_km'], 8);
    expect(backend.compareRequests.last['lat'], -5.07);
  });

  testWidgets('the selector shows the radius and changes it when the slider is let go', (tester) async {
    final backend = FakeBackend();
    final state = buildState(backend);
    await tester.pumpWidget(ChangeNotifierProvider<AppState>.value(
      value: state,
      child: const MaterialApp(home: Scaffold(body: RadiusSelector())),
    ));

    expect(find.text('até 5 km'), findsOneWidget);

    await tester.drag(find.byType(Slider), const Offset(300, 0));
    await tester.pumpAndSettle();

    expect(state.radiusKm, greaterThan(5));
    expect(find.text('até ${state.radiusKm.round()} km'), findsOneWidget);
    expect(_marketCalls(backend), isNotEmpty);
  });
}
