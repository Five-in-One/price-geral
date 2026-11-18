import 'dart:async' show Completer;

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:price/core/services/app_state.dart';
import 'package:price/core/services/location_service.dart';
import 'package:price/modules/search/search_page.dart';
import 'package:provider/provider.dart';

import 'comparison_test_helpers.dart';

class _Gps extends LocationService {
  final UserLocation? position;
  final LocationException? failure;
  int reads = 0;

  _Gps.at(double lat, double lng)
      : position = UserLocation(lat, lng),
        failure = null;

  _Gps.failing(this.failure) : position = null;

  @override
  Future<UserLocation> currentLocation() async {
    reads++;
    if (failure != null) throw failure!;
    return position!;
  }
}

const _button = ValueKey('update-location');

Future<AppState> pumpSearch(WidgetTester tester, FakeBackend backend, LocationService gps) async {
  tester.view.physicalSize = const Size(411, 900);
  tester.view.devicePixelRatio = 1.0;
  addTearDown(tester.view.reset);

  final state = buildState(backend, location: gps);
  state.setAddress('Teresina'); // what "Digitar endereço manualmente" leaves behind
  await tester.pumpWidget(
    ChangeNotifierProvider.value(
      value: state,
      child: const MaterialApp(home: Scaffold(body: SearchPage())),
    ),
  );
  await tester.pumpAndSettle();
  return state;
}

void main() {
  testWidgets('tapping the location at the top reads the GPS again and refreshes what depends on it', (tester) async {
    final backend = FakeBackend();
    final gps = _Gps.at(-5.07, -42.785);
    final state = await pumpSearch(tester, backend, gps);
    expect(find.text('Teresina'), findsOneWidget);
    final urlsBefore = backend.urls.length;

    await tester.tap(find.byKey(_button));
    await tester.pumpAndSettle();

    expect(gps.reads, 1);
    expect(state.userLocation!.lat, -5.07);
    expect(find.text('Minha localização'), findsOneWidget);
    expect(find.text('Localização atualizada.'), findsOneWidget);

    // the markets are asked again with the new position (distances depend on it)
    final marketCalls = backend.urls.skip(urlsBefore).where((u) => u.path == '/api/markets');
    expect(marketCalls, hasLength(1));
    expect(marketCalls.single.queryParameters['lat'], '-5.07');
    expect(marketCalls.single.queryParameters['lng'], '-42.785');
  });

  testWidgets('if the position cannot be read, says why and keeps what it had', (tester) async {
    final gps = _Gps.failing(const LocationException(
      LocationIssue.permissionBlocked,
      'A permissão de localização está bloqueada. Libere nas configurações do aparelho.',
    ));
    final state = await pumpSearch(tester, FakeBackend(), gps);

    await tester.tap(find.byKey(_button));
    await tester.pumpAndSettle();

    expect(find.textContaining('permissão de localização está bloqueada'), findsOneWidget);
    expect(find.text('Configurações'), findsOneWidget, reason: 'a shortcut to where the fix lives');
    expect(find.text('Teresina'), findsOneWidget, reason: 'the previous location is left alone');
    expect(state.isUpdatingLocation, isFalse);
    expect(find.text('Localização atualizada.'), findsNothing);
  });

  testWidgets('a retry is possible after a failure', (tester) async {
    final gps = _Gps.failing(const LocationException(LocationIssue.unavailable, 'Tente de novo.'));
    await pumpSearch(tester, FakeBackend(), gps);

    await tester.tap(find.byKey(_button));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(_button));
    await tester.pumpAndSettle();

    expect(gps.reads, 2);
  });

  testWidgets('the tap target is big enough for a finger', (tester) async {
    await pumpSearch(tester, FakeBackend(), _Gps.at(-5.07, -42.785));

    final size = tester.getSize(find.byKey(_button));
    expect(size.height, greaterThanOrEqualTo(40));
  });

  testWidgets('shows a spinner while the position is being read, and ignores taps meanwhile', (tester) async {
    final slow = _SlowGps();
    await pumpSearch(tester, FakeBackend(), slow);

    await tester.tap(find.byKey(_button));
    await tester.pump(); // GPS still pending
    expect(find.byType(CircularProgressIndicator), findsWidgets);

    await tester.tap(find.byKey(_button), warnIfMissed: false);
    await tester.pump();
    slow.finish(const UserLocation(-5.07, -42.785));
    await tester.pumpAndSettle();

    expect(slow.reads, 1, reason: 'a second tap while reading must not start another read');
  });
}

class _SlowGps extends LocationService {
  int reads = 0;
  final _done = Completer<UserLocation>();

  @override
  Future<UserLocation> currentLocation() {
    reads++;
    return _done.future;
  }

  void finish(UserLocation location) => _done.complete(location);
}
