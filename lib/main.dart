import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'core/services/app_state.dart';
import 'core/theme/app_theme.dart';
import 'splash.dart';

void main() {
  runApp(
    ChangeNotifierProvider(create: (_) => AppState()
        ..restoreCart()
        ..restoreRadius()
        ..restoreSession(), child: const MyApp()),
  );
}

class MyApp extends StatelessWidget {
  const MyApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Price App',
      debugShowCheckedModeBanner: false,
      theme: AppTheme.lightTheme,
      home: const SplashPage(),
    );
  }
}
