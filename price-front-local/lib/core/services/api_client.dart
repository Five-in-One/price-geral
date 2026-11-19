import 'dart:convert';
import 'dart:io' show Platform;

import 'package:flutter/foundation.dart' show debugPrint, kIsWeb;
import 'package:http/http.dart' as http;

/// Thrown when the API returns a non-2xx response or the request fails.
class ApiException implements Exception {
  final String message;

  /// The HTTP status, or null when the request never got an answer.
  final int? statusCode;

  /// The API's machine-readable error code (`username_taken`,
  /// `product_not_in_catalog`...); null when the server sent none.
  final String? code;

  ApiException(this.message, {this.statusCode, this.code});

  /// The server rejected the login token (missing, expired or invalid).
  bool get isUnauthorized => statusCode == 401;

  @override
  String toString() => message;
}

/// Thin JSON HTTP wrapper around the Price API: the production API (one
/// base URL) or, for local development, one of the backend services
/// (price-back, price-auth, price-user — each is its own process/port).
class ApiClient {
  final http.Client _client;
  final String baseUrl;

  ApiClient({required this.baseUrl, http.Client? client}) : _client = client ?? http.Client();

  /// The production API, which serves every route under one base URL.
  /// Override it with `--dart-define=API_BASE_URL=<url>`.
  static const String productionUrl = String.fromEnvironment(
    'API_BASE_URL',
    defaultValue: 'https://api.pricebrasil.com',
  );

  /// Host of the local backend services. This copy of the app (price-front-local)
  /// always talks to them; by default through the dev machine's LAN IP, which
  /// works from a physical device on the same Wi-Fi and from the Android
  /// emulator. Override it with `--dart-define=API_HOST=<ip>` when the machine's
  /// IP changes (`10.0.2.2` for an Android emulator, `localhost` elsewhere), or
  /// with an empty value to fall back to the platform default below.
  static const String _hostOverride = String.fromEnvironment('API_HOST', defaultValue: '192.168.18.138');

  /// Whether the app runs against the local services instead of production.
  /// Always true in price-front-local.
  static bool get usesLocalServices => true;

  /// Whether the local services are reached through the Nginx gateway
  /// (docker compose: one port for everything, price-back load-balanced over
  /// 3 replicas). Pass `--dart-define=API_GATEWAY=false` to talk to each
  /// service on its own port (8080/8081/8082) when running them with `go run`.
  static const bool viaGateway = bool.fromEnvironment('API_GATEWAY', defaultValue: true);

  /// Port the gateway listens on (`--dart-define=API_GATEWAY_PORT=<port>`).
  static const int gatewayPort = int.fromEnvironment('API_GATEWAY_PORT', defaultValue: 80);

  /// Base URL on the gateway. [prefix] is `/api` for price-auth, whose
  /// `/auth/...` routes the gateway publishes as `/api/auth/...`.
  static String gatewayUrl([String prefix = '']) => '${hostUrlForPort(gatewayPort)}$prefix';

  /// Resolves the base URL for a local service running on [port] on the host
  /// machine. With [_hostOverride] set, every service uses that host.
  /// Otherwise Android emulators reach the host through 10.0.2.2 instead of
  /// localhost, and every other target (web, iOS simulator, desktop) can use
  /// localhost directly.
  static String hostUrlForPort(int port) {
    if (_hostOverride.isNotEmpty) return 'http://$_hostOverride:$port';
    if (kIsWeb) return 'http://localhost:$port';
    try {
      if (Platform.isAndroid) return 'http://10.0.2.2:$port';
    } catch (_) {
      // Platform is unavailable in some non-mobile/non-web test contexts.
    }
    return 'http://localhost:$port';
  }

  Uri _uri(String path, [Map<String, String>? query]) {
    return Uri.parse('$baseUrl$path').replace(queryParameters: (query == null || query.isEmpty) ? null : query);
  }

  /// [token] is a price-auth access token, sent as `Authorization: Bearer`.
  /// It is never logged.
  Future<dynamic> get(String path, {Map<String, String>? query, String? token}) async {
    final uri = _uri(path, query);
    final http.Response res;
    final stopwatch = Stopwatch()..start();
    debugPrint('[API] --> GET $uri');
    try {
      res = await _client.get(uri, headers: _authHeaders(token)).timeout(const Duration(seconds: 10));
    } catch (e) {
      debugPrint('[API] xxx GET $uri falhou em ${stopwatch.elapsedMilliseconds}ms: ${e.runtimeType}: $e');
      throw ApiException('Não foi possível conectar ao servidor.');
    }
    _logResponse('GET', uri, res, stopwatch);
    return _decode(res);
  }

  Future<dynamic> post(String path, {Object? body, String? token}) async {
    final uri = _uri(path);
    final http.Response res;
    final stopwatch = Stopwatch()..start();
    // The request body is not logged: login/register payloads carry passwords.
    debugPrint('[API] --> POST $uri');
    try {
      res = await _client
          .post(uri, headers: {'Content-Type': 'application/json', ..._authHeaders(token)}, body: jsonEncode(body))
          .timeout(const Duration(seconds: 10));
    } catch (e) {
      debugPrint('[API] xxx POST $uri falhou em ${stopwatch.elapsedMilliseconds}ms: ${e.runtimeType}: $e');
      throw ApiException('Não foi possível conectar ao servidor.');
    }
    _logResponse('POST', uri, res, stopwatch);
    return _decode(res);
  }

  Future<dynamic> delete(String path, {String? token}) async {
    final uri = _uri(path);
    final http.Response res;
    final stopwatch = Stopwatch()..start();
    debugPrint('[API] --> DELETE $uri');
    try {
      res = await _client.delete(uri, headers: _authHeaders(token)).timeout(const Duration(seconds: 10));
    } catch (e) {
      debugPrint('[API] xxx DELETE $uri falhou em ${stopwatch.elapsedMilliseconds}ms: ${e.runtimeType}: $e');
      throw ApiException('Não foi possível conectar ao servidor.');
    }
    _logResponse('DELETE', uri, res, stopwatch);
    return _decode(res);
  }

  Map<String, String> _authHeaders(String? token) => token == null ? const {} : {'Authorization': 'Bearer $token'};

  /// Logs status, latency and size. The body is only printed for non-2xx
  /// responses (truncated): successful bodies can hold auth tokens.
  void _logResponse(String method, Uri uri, http.Response res, Stopwatch stopwatch) {
    final ok = res.statusCode >= 200 && res.statusCode < 300;
    debugPrint(
      '[API] <-- ${res.statusCode} $method $uri (${stopwatch.elapsedMilliseconds}ms, ${res.bodyBytes.length}B)',
    );
    if (!ok) {
      final body = res.body;
      debugPrint('[API]     corpo: ${body.length > 500 ? '${body.substring(0, 500)}…' : body}');
    }
  }

  dynamic _decode(http.Response res) {
    if (res.statusCode < 200 || res.statusCode >= 300) {
      String message = 'Erro ${res.statusCode} ao falar com o servidor.';
      String? code;
      try {
        final decoded = jsonDecode(res.body);
        final error = decoded is Map ? decoded['error'] : null;
        if (error is String) {
          // Local services: {"error": "message"}.
          message = error;
        } else if (error is Map) {
          // Production: {"error": {"code": "...", "message": "..."}}.
          if (error['message'] is String) message = error['message'] as String;
          if (error['code'] is String) code = error['code'] as String;
        }
      } catch (_) {
        // Body wasn't JSON; keep the generic message.
      }
      throw ApiException(message, statusCode: res.statusCode, code: code);
    }
    if (res.body.isEmpty) return null;
    return jsonDecode(res.body);
  }
}
