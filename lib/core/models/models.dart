import 'package:flutter/material.dart';

/// Parses a "#RRGGBB" hex string, as returned by the API, into a [Color].
Color colorFromHex(String hexColor) {
  final buffer = StringBuffer();
  if (hexColor.length == 6 || hexColor.length == 7) buffer.write('ff');
  buffer.write(hexColor.replaceFirst('#', ''));
  return Color(int.parse(buffer.toString(), radix: 16));
}

/// Formats an amount in reais for display, e.g. `R$ 12,50`.
String formatBrl(double value) => 'R\$ ${value.toStringAsFixed(2).replaceAll('.', ',')}';

/// Formats a distance for display, e.g. `1,2 km`.
String formatKm(double km) {
  if (km < 0.1) return 'menos de 0,1 km';
  return '${km.toStringAsFixed(1).replaceAll('.', ',')} km';
}

class Market {
  final String id;
  final String name;
  final String initials;
  final double? distance; // in km; null when unknown (no location on either side)
  final Color primaryColor;
  final Color backgroundColor;

  // Address; any piece is "" when the market didn't register it.
  final String street;
  final String number;
  final String complement;
  final String neighborhood;
  final String city;
  final String state;
  final String cep;

  const Market({
    required this.id,
    required this.name,
    required this.initials,
    this.distance,
    required this.primaryColor,
    required this.backgroundColor,
    this.street = '',
    this.number = '',
    this.complement = '',
    this.neighborhood = '',
    this.city = '',
    this.state = '',
    this.cep = '',
  });

  /// "Rua Félix Pacheco, 965 - Loja 2"; "" when the street is unknown.
  String get streetLine {
    if (street.isEmpty) return '';
    final withNumber = number.isEmpty ? street : '$street, $number';
    return complement.isEmpty ? withNumber : '$withNumber - $complement';
  }

  /// "Centro Sul, Teresina - PI".
  String get regionLine {
    final cityState = [city, state].where((s) => s.isNotEmpty).join(' - ');
    return [neighborhood, cityState].where((s) => s.isNotEmpty).join(', ');
  }

  /// The whole address on one line, e.g. for a map search; "" when unknown.
  String get fullAddress {
    final parts = [streetLine, regionLine, if (cep.isNotEmpty) 'CEP $cep'];
    return parts.where((s) => s.isNotEmpty).join(', ');
  }

  bool get hasAddress => streetLine.isNotEmpty || regionLine.isNotEmpty;

  /// Builds a Market from a GET /api/markets entry. The nested `market`
  /// object returned by POST /api/comparisons has no `distance_km`, and a
  /// market without a registered location has `distance_km: null`; both
  /// leave [distance] null so no fake distance is shown.
  factory Market.fromJson(Map<String, dynamic> json) {
    return Market(
      id: json['id'] as String,
      name: json['name'] as String,
      initials: json['initials'] as String,
      distance: (json['distance_km'] as num?)?.toDouble(),
      primaryColor: colorFromHex(json['primary_color'] as String),
      backgroundColor: colorFromHex(json['background_color'] as String),
      // The API leaves out the address pieces the market didn't fill in.
      street: _text(json['street']),
      number: _text(json['number']),
      complement: _text(json['complement']),
      neighborhood: _text(json['neighborhood']),
      city: _text(json['city']),
      state: _text(json['state']),
      cep: _text(json['cep']),
    );
  }

  static String _text(Object? value) => value is String ? value.trim() : '';
}

class Product {
  final String id;
  final String name;
  final String unit; // "kg", "un", "pct", etc.
  final String category; // "GRÃOS", "LATICÍNIOS", "CARNES", "HORTIFRUTI"
  final String defaultMarketName; // Primary market showing on browse page
  final double defaultPrice; // Price shown on browse page

  /// The canonical catalog item this listing is an offer of. Two markets'
  /// listings of the same product share it, which is what lets a shopping
  /// list be priced across markets. Null for listings that aren't in the
  /// catalog (those can't be compared).
  final String? catalogId;
  final String brand; // "" when unknown
  final String sizeLabel; // "500 g", "12 x 30 m"; "" when unknown
  final double? unitPrice; // price per kg / L / m / un
  final String unitPriceUnit;

  /// How many markets sell this product; the search groups the listings of one
  /// catalog item into a single row.
  final int marketsCount;

  const Product({
    required this.id,
    required this.name,
    required this.unit,
    required this.category,
    required this.defaultMarketName,
    required this.defaultPrice,
    this.catalogId,
    this.brand = '',
    this.sizeLabel = '',
    this.unitPrice,
    this.unitPriceUnit = '',
    this.marketsCount = 1,
  });

  factory Product.fromJson(Map<String, dynamic> json) {
    final catalogId = json['catalog_id'] as String?;
    return Product(
      id: json['id'] as String,
      name: json['name'] as String,
      unit: json['unit'] as String,
      category: json['category'] as String,
      defaultMarketName: json['default_market_name'] as String,
      defaultPrice: (json['default_price'] as num).toDouble(),
      catalogId: (catalogId == null || catalogId.isEmpty) ? null : catalogId,
      brand: (json['brand'] as String?) ?? '',
      sizeLabel: (json['size_label'] as String?) ?? '',
      unitPrice: (json['unit_price'] as num?)?.toDouble(),
      unitPriceUnit: (json['unit_price_unit'] as String?) ?? '',
    );
  }

  /// Same keys as the API, so [Product.fromJson] reads it back (the shopping
  /// list is saved on the device).
  Map<String, dynamic> toJson() => {
        'id': id,
        'name': name,
        'unit': unit,
        'category': category,
        'default_market_name': defaultMarketName,
        'default_price': defaultPrice,
        'catalog_id': catalogId,
        'brand': brand,
        'size_label': sizeLabel,
        'unit_price': unitPrice,
        'unit_price_unit': unitPriceUnit,
      };

  Product copyWith({double? defaultPrice, String? defaultMarketName, int? marketsCount}) {
    return Product(
      id: id,
      name: name,
      unit: unit,
      category: category,
      defaultMarketName: defaultMarketName ?? this.defaultMarketName,
      defaultPrice: defaultPrice ?? this.defaultPrice,
      catalogId: catalogId,
      brand: brand,
      sizeLabel: sizeLabel,
      unitPrice: unitPrice,
      unitPriceUnit: unitPriceUnit,
      marketsCount: marketsCount ?? this.marketsCount,
    );
  }
}

class CartItem {
  final Product product;
  int quantity;

  CartItem({
    required this.product,
    required this.quantity,
  });

  double get totalPrice => product.defaultPrice * quantity;

  Map<String, dynamic> toJson() => {'product': product.toJson(), 'quantity': quantity};

  factory CartItem.fromJson(Map<String, dynamic> json) => CartItem(
        product: Product.fromJson(json['product'] as Map<String, dynamic>),
        quantity: json['quantity'] as int,
      );
}

class SavingsHistoryItem {
  final String dateLabel; // "24 MAI"
  final String marketName;
  final int itemsCount;
  final double totalCost;
  final double savings;

  const SavingsHistoryItem({
    required this.dateLabel,
    required this.marketName,
    required this.itemsCount,
    required this.totalCost,
    required this.savings,
  });

  factory SavingsHistoryItem.fromJson(Map<String, dynamic> json) {
    return SavingsHistoryItem(
      dateLabel: json['date_label'] as String,
      marketName: json['market_name'] as String,
      itemsCount: json['items_count'] as int,
      totalCost: (json['total_cost'] as num).toDouble(),
      savings: (json['savings'] as num).toDouble(),
    );
  }
}

/// One bar of the "savings per month" chart on the Economy tab.
class MonthlyPoint {
  final String label;
  final double value;
  final bool isActive;

  const MonthlyPoint({required this.label, required this.value, required this.isActive});

  factory MonthlyPoint.fromJson(Map<String, dynamic> json) {
    return MonthlyPoint(
      label: json['label'] as String,
      value: (json['value'] as num).toDouble(),
      isActive: json['is_active'] as bool,
    );
  }
}

/// Headline numbers and chart series shown on the Economy tab.
class EconomySummary {
  final double totalSaved;
  final String monthLabel;
  final double monthSpend;
  final double monthSavings;
  final double monthSavingsPercent;
  final List<MonthlyPoint> monthlyChart;

  const EconomySummary({
    required this.totalSaved,
    required this.monthLabel,
    required this.monthSpend,
    required this.monthSavings,
    required this.monthSavingsPercent,
    required this.monthlyChart,
  });

  factory EconomySummary.fromJson(Map<String, dynamic> json) {
    return EconomySummary(
      totalSaved: (json['total_saved'] as num).toDouble(),
      monthLabel: json['month_label'] as String,
      monthSpend: (json['month_spend'] as num).toDouble(),
      monthSavings: (json['month_savings'] as num).toDouble(),
      monthSavingsPercent: (json['month_savings_percent'] as num).toDouble(),
      monthlyChart: (json['monthly_chart'] as List<dynamic>)
          .map((e) => MonthlyPoint.fromJson(e as Map<String, dynamic>))
          .toList(),
    );
  }
}
