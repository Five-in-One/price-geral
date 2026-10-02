import 'models.dart';

/// The person signed in to the app (price-auth's user, minus what the app
/// doesn't need).
class AppUser {
  final String id;
  final String username;

  const AppUser({required this.id, required this.username});

  factory AppUser.fromJson(Map<String, dynamic> json) => AppUser(
        id: json['id'] as String,
        username: json['username'] as String,
      );

  Map<String, dynamic> toJson() => {'id': id, 'username': username};
}

/// A signed-in user plus the token that proves it to price-auth/price-user.
class AuthSession {
  final String token;
  final AppUser user;

  const AuthSession({required this.token, required this.user});

  /// Reads price-auth's login/signup response.
  factory AuthSession.fromAuthResponse(Map<String, dynamic> json) => AuthSession(
        token: json['access_token'] as String,
        user: AppUser.fromJson(json['user'] as Map<String, dynamic>),
      );
}

/// A shopping list the person saved to their account.
class SavedList {
  final String id;
  final String name;
  final List<CartItem> items;
  final DateTime updatedAt;

  const SavedList({
    required this.id,
    required this.name,
    required this.items,
    required this.updatedAt,
  });

  int get itemCount => items.length;

  /// Units across every item ("2 arroz + 1 feijão" is 3).
  int get totalQuantity => items.fold(0, (sum, item) => sum + item.quantity);

  factory SavedList.fromJson(Map<String, dynamic> json) => SavedList(
        id: json['id'] as String,
        name: json['name'] as String,
        items: (json['items'] as List<dynamic>)
            .map((e) => CartItem.fromJson(e as Map<String, dynamic>))
            .toList(),
        updatedAt: DateTime.parse(json['updated_at'] as String).toLocal(),
      );
}

/// The street address of the device's position, from price-back's reverse
/// geocoding. Any piece can be empty when the map doesn't know it.
class PlaceAddress {
  final String label;
  final String city;
  final String state;

  const PlaceAddress({required this.label, required this.city, required this.state});

  factory PlaceAddress.fromJson(Map<String, dynamic> json) => PlaceAddress(
        label: (json['label'] as String?) ?? '',
        city: (json['city'] as String?) ?? '',
        state: (json['state'] as String?) ?? '',
      );
}
