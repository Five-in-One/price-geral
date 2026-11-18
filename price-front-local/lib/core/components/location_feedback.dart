import 'package:flutter/material.dart';

import '../services/app_state.dart';
import '../services/location_service.dart';

/// Tells the person why their position couldn't be read, with a shortcut to the
/// system settings when that is where the fix lives (GPS off, permission
/// blocked for good). Takes the messenger up front so it stays valid across the
/// `await` that precedes it.
void showLocationError(ScaffoldMessengerState messenger, AppState appState, LocationException e) {
  messenger.showSnackBar(
    SnackBar(
      content: Text(e.message),
      duration: const Duration(seconds: 6),
      action: e.needsSettings
          ? SnackBarAction(
              label: 'Configurações',
              onPressed: () => appState.openLocationSettings(e.issue),
            )
          : null,
    ),
  );
}
