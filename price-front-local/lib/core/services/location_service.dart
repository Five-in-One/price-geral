import 'package:flutter/foundation.dart' show debugPrint;
import 'package:geolocator/geolocator.dart';

/// The user's position, as read from the device GPS.
class UserLocation {
  final double lat;
  final double lng;
  const UserLocation(this.lat, this.lng);
}

enum LocationIssue {
  /// GPS / location services are switched off on the device.
  serviceOff,

  /// The user refused the permission prompt (can be asked again).
  permissionDenied,

  /// The permission is blocked for good; only the system settings can fix it.
  permissionBlocked,

  /// Permission is fine but no position could be read.
  unavailable,
}

/// Thrown with a [message] that is safe to show to the user.
class LocationException implements Exception {
  final LocationIssue issue;
  final String message;
  const LocationException(this.issue, this.message);

  /// Whether the fix lives in the system settings rather than a retry.
  bool get needsSettings => issue == LocationIssue.serviceOff || issue == LocationIssue.permissionBlocked;

  @override
  String toString() => message;
}

/// Reads the device position through the OS location services.
class LocationService {
  const LocationService();

  /// Asks for permission if needed and returns the current position.
  /// Throws [LocationException] when it can't be obtained.
  Future<UserLocation> currentLocation() async {
    if (!await Geolocator.isLocationServiceEnabled()) {
      throw const LocationException(
        LocationIssue.serviceOff,
        'A localização do aparelho está desligada. Ative o GPS e tente de novo.',
      );
    }

    var permission = await Geolocator.checkPermission();
    if (permission == LocationPermission.denied) {
      permission = await Geolocator.requestPermission();
    }
    if (permission == LocationPermission.denied) {
      throw const LocationException(
        LocationIssue.permissionDenied,
        'Sem permissão de localização. Permita o acesso para ver a distância até os mercados.',
      );
    }
    if (permission == LocationPermission.deniedForever) {
      throw const LocationException(
        LocationIssue.permissionBlocked,
        'A permissão de localização está bloqueada. Libere nas configurações do aparelho.',
      );
    }

    try {
      final position = await Geolocator.getCurrentPosition(
        locationSettings: const LocationSettings(accuracy: LocationAccuracy.high, timeLimit: Duration(seconds: 15)),
      );
      debugPrint('[LOC] ${position.latitude}, ${position.longitude} (±${position.accuracy.round()}m)');
      return UserLocation(position.latitude, position.longitude);
    } catch (e) {
      debugPrint('[LOC] falhou: ${e.runtimeType}: $e');
      throw const LocationException(
        LocationIssue.unavailable,
        'Não foi possível obter sua localização agora. Tente de novo.',
      );
    }
  }

  /// Opens the system screen where the user can fix [issue].
  Future<void> openSettings(LocationIssue issue) async {
    if (issue == LocationIssue.serviceOff) {
      await Geolocator.openLocationSettings();
    } else {
      await Geolocator.openAppSettings();
    }
  }
}
