import 'dart:convert';

import 'package:flutter/foundation.dart' show debugPrint;
import 'package:shared_preferences/shared_preferences.dart';

import '../models/account.dart';

/// Keeps the signed-in session between app launches.
abstract class AuthStorage {
  Future<AuthSession?> load();
  Future<void> save(AuthSession session);
  Future<void> clear();
}

/// Stores the session on the device (SharedPreferences), as JSON.
class SharedPreferencesAuthStorage implements AuthStorage {
  static const _key = 'auth_session_v1';

  const SharedPreferencesAuthStorage();

  @override
  Future<AuthSession?> load() async {
    try {
      final raw = (await SharedPreferences.getInstance()).getString(_key);
      if (raw == null || raw.isEmpty) return null;
      final json = jsonDecode(raw) as Map<String, dynamic>;
      return AuthSession(
        token: json['token'] as String,
        user: AppUser.fromJson(json['user'] as Map<String, dynamic>),
      );
    } catch (e) {
      // A session saved by an older/incompatible version just means "signed out".
      debugPrint('[AUTH] não foi possível ler a sessão salva: $e');
      return null;
    }
  }

  @override
  Future<void> save(AuthSession session) async {
    try {
      final prefs = await SharedPreferences.getInstance();
      await prefs.setString(_key, jsonEncode({'token': session.token, 'user': session.user.toJson()}));
    } catch (e) {
      debugPrint('[AUTH] não foi possível salvar a sessão: $e');
    }
  }

  @override
  Future<void> clear() async {
    try {
      await (await SharedPreferences.getInstance()).remove(_key);
    } catch (e) {
      debugPrint('[AUTH] não foi possível apagar a sessão: $e');
    }
  }
}

/// In-memory storage, for tests.
class MemoryAuthStorage implements AuthStorage {
  AuthSession? saved;

  MemoryAuthStorage([this.saved]);

  @override
  Future<AuthSession?> load() async => saved;

  @override
  Future<void> save(AuthSession session) async => saved = session;

  @override
  Future<void> clear() async => saved = null;
}
