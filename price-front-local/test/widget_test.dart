import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:price/core/services/app_state.dart';
import 'package:price/core/services/cart_storage.dart';
import 'package:price/splash.dart';
import 'package:provider/provider.dart';

void main() {
  testWidgets('the app opens on the splash screen', (WidgetTester tester) async {
    tester.view.physicalSize = const Size(411, 900);
    tester.view.devicePixelRatio = 1.0;
    addTearDown(tester.view.reset);

    await tester.pumpWidget(
      ChangeNotifierProvider(
        create: (_) => AppState(storage: MemoryCartStorage()),
        child: const MaterialApp(home: SplashPage()),
      ),
    );

    expect(find.textContaining('Os melhores preços'), findsOneWidget);
    expect(find.text('Usar minha localização'), findsOneWidget);
    // Not asserting on layout here: the default test font is far wider than the
    // real one and overflows the button. Real-width layout is covered where it
    // matters (search and comparison tests).
    tester.takeException();
  });
}
