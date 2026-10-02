import 'package:flutter/foundation.dart' show debugPrint;
import 'package:shared_preferences/shared_preferences.dart';

/// Keeps the person's preferences between app launches.
abstract class SettingsStorage {
  /// The markets search radius in km, or null when never chosen.
  Future<double?> loadRadiusKm();
  Future<void> saveRadiusKm(double km);
}

/// Stores the preferences on the device (SharedPreferences).
class SharedPreferencesSettingsStorage implements SettingsStorage {
  static const _radiusKey = 'markets_radius_km_v1';

  const SharedPreferencesSettingsStorage();

  @override
  Future<double?> loadRadiusKm() async {
    try {
      return (await SharedPreferences.getInstance()).getDouble(_radiusKey);
    } catch (e) {
      debugPrint('[SETTINGS] não foi possível ler o raio salvo: $e');
      return null;
    }
  }

  @override
  Future<void> saveRadiusKm(double km) async {
    try {
      await (await SharedPreferences.getInstance()).setDouble(_radiusKey, km);
    } catch (e) {
      debugPrint('[SETTINGS] não foi possível salvar o raio: $e');
    }
  }
}

/// In-memory storage, for tests.
class MemorySettingsStorage implements SettingsStorage {
  double? radiusKm;

  MemorySettingsStorage([this.radiusKm]);

  @override
  Future<double?> loadRadiusKm() async => radiusKm;

  @override
  Future<void> saveRadiusKm(double km) async => radiusKm = km;
}
