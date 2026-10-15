import 'models.dart';

/// Orders the product search accepts (`sort` on GET /api/products/search).
enum ProductSort {
  priceAsc('price_asc', 'Menor preço'),
  priceDesc('price_desc', 'Maior preço'),
  unitPriceAsc('unit_price_asc', 'Menor preço por kg/L'),
  nameAsc('name_asc', 'Nome (A-Z)'),
  marketsDesc('markets_desc', 'Em mais mercados');

  final String apiValue;
  final String label;
  const ProductSort(this.apiValue, this.label);
}

/// Everything that narrows the product search besides the typed text and the
/// category. Applied by the server, never on the device.
class SearchFilters {
  /// Only offers from these markets; empty means every market.
  final Set<String> marketIds;
  final double? minPrice;
  final double? maxPrice;

  /// Only markets within the person's search radius (needs a position).
  final bool nearbyOnly;

  const SearchFilters({this.marketIds = const {}, this.minPrice, this.maxPrice, this.nearbyOnly = false});

  /// How many filters are on, for the badge on the "Filtros" button.
  int get activeCount =>
      (marketIds.isEmpty ? 0 : 1) + (minPrice != null || maxPrice != null ? 1 : 0) + (nearbyOnly ? 1 : 0);

  bool get isEmpty => activeCount == 0;
}

/// One page of the grouped product search: one row per product (its cheapest
/// offer, with how many markets sell it).
class ProductPage {
  final List<Product> items;
  final int total;
  final bool hasMore;

  const ProductPage({required this.items, required this.total, required this.hasMore});

  factory ProductPage.fromJson(dynamic json) {
    // A plain list is a server without paging: everything in one page.
    if (json is List) {
      final items = json.map((e) => Product.fromJson(e as Map<String, dynamic>)).toList();
      return ProductPage(items: items, total: items.length, hasMore: false);
    }
    final map = json as Map<String, dynamic>;
    final items = (map['items'] as List<dynamic>? ?? const [])
        .map((e) => Product.fromJson(e as Map<String, dynamic>))
        .toList();
    return ProductPage(
      items: items,
      total: (map['total'] as num?)?.toInt() ?? items.length,
      hasMore: map['has_more'] as bool? ?? false,
    );
  }
}

/// A product category with how many search rows it holds.
class CategoryCount {
  final String category; // as the server stores it: "HORTIFRUTI"
  final int count;

  const CategoryCount(this.category, this.count);

  factory CategoryCount.fromJson(Map<String, dynamic> json) =>
      CategoryCount(json['category'] as String, (json['count'] as num).toInt());

  /// "LATICÍNIOS" -> "Laticínios".
  String get label => category.isEmpty ? category : category[0] + category.substring(1).toLowerCase();
}
