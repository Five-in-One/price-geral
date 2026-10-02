import 'dart:convert';

import 'package:flutter/foundation.dart' show debugPrint;
import 'package:shared_preferences/shared_preferences.dart';

import '../models/models.dart';

/// Keeps the shopping list between app launches.
abstract class CartStorage {
  Future<List<CartItem>> load();
  Future<void> save(List<CartItem> items);
}

/// Stores the list on the device (SharedPreferences), as JSON.
class SharedPreferencesCartStorage implements CartStorage {
  static const _key = 'shopping_list_v1';

  const SharedPreferencesCartStorage();

  @override
  Future<List<CartItem>> load() async {
    try {
      final raw = (await SharedPreferences.getInstance()).getString(_key);
      if (raw == null || raw.isEmpty) return [];
      return (jsonDecode(raw) as List<dynamic>)
          .map((e) => CartItem.fromJson(e as Map<String, dynamic>))
          .toList();
    } catch (e) {
      // A list saved by an older/incompatible version must not break the app.
      debugPrint('[CART] não foi possível ler a lista salva: $e');
      return [];
    }
  }

  @override
  Future<void> save(List<CartItem> items) async {
    try {
      final prefs = await SharedPreferences.getInstance();
      await prefs.setString(_key, jsonEncode(items.map((i) => i.toJson()).toList()));
    } catch (e) {
      debugPrint('[CART] não foi possível salvar a lista: $e');
    }
  }
}

/// In-memory storage, for tests.
class MemoryCartStorage implements CartStorage {
  List<CartItem> saved;

  MemoryCartStorage([List<CartItem>? initial]) : saved = initial ?? [];

  @override
  Future<List<CartItem>> load() async => List.of(saved);

  @override
  Future<void> save(List<CartItem> items) async {
    saved = items.map((i) => CartItem(product: i.product, quantity: i.quantity)).toList();
  }
}
