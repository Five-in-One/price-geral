import 'models.dart';

/// One list item as one market would sell it.
class ListLine {
  final String catalogId;
  final String name; // the item the user asked for
  final int quantity;
  final String productId;
  final String productName; // what this market actually lists
  final String brand;
  final String sizeLabel;
  final double price;
  final double lineTotal;
  final double? unitPrice;
  final String unitPriceUnit;

  /// Another brand of the requested product (the market lacks the one asked for).
  final bool isSubstitute;

  /// No market sells this item for less.
  final bool isCheapest;

  const ListLine({
    required this.catalogId,
    required this.name,
    required this.quantity,
    required this.productId,
    required this.productName,
    required this.price,
    required this.lineTotal,
    this.brand = '',
    this.sizeLabel = '',
    this.unitPrice,
    this.unitPriceUnit = '',
    this.isSubstitute = false,
    this.isCheapest = false,
  });

  factory ListLine.fromJson(Map<String, dynamic> json) {
    return ListLine(
      catalogId: json['catalog_id'] as String,
      name: json['name'] as String,
      quantity: json['quantity'] as int,
      productId: json['product_id'] as String,
      productName: json['product_name'] as String,
      brand: (json['brand'] as String?) ?? '',
      sizeLabel: (json['size_label'] as String?) ?? '',
      price: (json['price'] as num).toDouble(),
      lineTotal: (json['line_total'] as num).toDouble(),
      unitPrice: (json['unit_price'] as num?)?.toDouble(),
      unitPriceUnit: (json['unit_price_unit'] as String?) ?? '',
      isSubstitute: (json['is_substitute'] as bool?) ?? false,
      isCheapest: (json['is_cheapest'] as bool?) ?? false,
    );
  }
}

/// A list item a market doesn't sell.
class MissingItem {
  final String catalogId;
  final String name;
  final int quantity;

  const MissingItem({required this.catalogId, required this.name, required this.quantity});

  factory MissingItem.fromJson(Map<String, dynamic> json) => MissingItem(
        catalogId: json['catalog_id'] as String,
        name: json['name'] as String,
        quantity: json['quantity'] as int,
      );
}

/// What the whole list costs at one market.
class MarketComparison {
  final Market market;
  final double? distanceKm;
  final double totalCost;

  /// The market has *every* item of the list. Only complete markets are
  /// comparable on price: an incomplete one looks cheap just because things
  /// are missing.
  final bool isComplete;
  final int foundItems;
  final int totalItems;
  final int substitutesCount;

  /// More than the cheapest complete market; 0 for incomplete ones.
  final double difference;
  final List<ListLine> lines;
  final List<MissingItem> missing;

  const MarketComparison({
    required this.market,
    required this.totalCost,
    required this.isComplete,
    required this.foundItems,
    required this.totalItems,
    this.distanceKm,
    this.substitutesCount = 0,
    this.difference = 0,
    this.lines = const [],
    this.missing = const [],
  });

  factory MarketComparison.fromJson(Map<String, dynamic> json) {
    return MarketComparison(
      market: Market.fromJson(json['market'] as Map<String, dynamic>),
      distanceKm: (json['distance_km'] as num?)?.toDouble(),
      totalCost: (json['total_cost'] as num).toDouble(),
      isComplete: json['is_complete'] as bool,
      foundItems: json['found_items'] as int,
      totalItems: json['total_items'] as int,
      substitutesCount: (json['substitutes_count'] as int?) ?? 0,
      difference: (json['difference'] as num?)?.toDouble() ?? 0,
      lines: _list(json['lines'], ListLine.fromJson),
      missing: _list(json['missing'], MissingItem.fromJson),
    );
  }

  int get missingCount => totalItems - foundItems;
}

/// A requested item across all markets.
class ListItemSummary {
  final String catalogId;
  final String name;
  final int quantity;

  /// At least one market sells it (or an accepted substitute).
  final bool available;
  final String? cheapestMarketId;
  final double? cheapestPrice;

  const ListItemSummary({
    required this.catalogId,
    required this.name,
    required this.quantity,
    required this.available,
    this.cheapestMarketId,
    this.cheapestPrice,
  });

  factory ListItemSummary.fromJson(Map<String, dynamic> json) {
    final cheapest = json['cheapest_market_id'] as String?;
    return ListItemSummary(
      catalogId: json['catalog_id'] as String,
      name: json['name'] as String,
      quantity: json['quantity'] as int,
      available: json['available'] as bool,
      cheapestMarketId: (cheapest == null || cheapest.isEmpty) ? null : cheapest,
      cheapestPrice: (json['cheapest_price'] as num?)?.toDouble(),
    );
  }
}

/// One market visited in the best combination.
class Stop {
  final Market market;
  final double? distanceKm;
  final double subtotal;
  final List<ListLine> lines;

  const Stop({required this.market, required this.subtotal, this.distanceKm, this.lines = const []});

  factory Stop.fromJson(Map<String, dynamic> json) => Stop(
        market: Market.fromJson(json['market'] as Map<String, dynamic>),
        distanceKm: (json['distance_km'] as num?)?.toDouble(),
        subtotal: (json['subtotal'] as num).toDouble(),
        lines: _list(json['lines'], ListLine.fromJson),
      );
}

/// Every item bought where it is cheapest, even if that takes several markets.
/// The server does not weigh the trip: showing [stops] next to
/// [savingsVsBestSingle] lets the person decide whether it is worth it.
class BestCombination {
  final double totalCost;

  /// Cheaper than the best single market by this much; null when no single
  /// market has the whole list.
  final double? savingsVsBestSingle;
  final List<Stop> stops;

  const BestCombination({required this.totalCost, required this.stops, this.savingsVsBestSingle});

  factory BestCombination.fromJson(Map<String, dynamic> json) => BestCombination(
        totalCost: (json['total_cost'] as num).toDouble(),
        savingsVsBestSingle: (json['savings_vs_best_single'] as num?)?.toDouble(),
        stops: _list(json['stops'], Stop.fromJson),
      );
}

/// The answer to POST /api/comparisons/list.
class ListComparison {
  final bool allowEquivalents;
  final List<ListItemSummary> items;

  /// Ranked: markets with the whole list first (cheapest first), then the
  /// incomplete ones, most items first.
  final List<MarketComparison> markets;
  final String? bestSingleMarketId;
  final BestCombination? bestCombination;

  /// Cheapest vs most expensive market *among the complete ones*.
  final double maxSavings;
  final double maxSavingsPercent;

  const ListComparison({
    required this.allowEquivalents,
    required this.items,
    required this.markets,
    this.bestSingleMarketId,
    this.bestCombination,
    this.maxSavings = 0,
    this.maxSavingsPercent = 0,
  });

  factory ListComparison.fromJson(Map<String, dynamic> json) {
    final best = json['best_single_market_id'] as String?;
    final combo = json['best_combination'] as Map<String, dynamic>?;
    return ListComparison(
      allowEquivalents: (json['allow_equivalents'] as bool?) ?? false,
      items: _list(json['items'], ListItemSummary.fromJson),
      markets: _list(json['markets'], MarketComparison.fromJson),
      bestSingleMarketId: (best == null || best.isEmpty) ? null : best,
      bestCombination: combo == null ? null : BestCombination.fromJson(combo),
      maxSavings: (json['max_savings'] as num?)?.toDouble() ?? 0,
      maxSavingsPercent: (json['max_savings_percent'] as num?)?.toDouble() ?? 0,
    );
  }

  /// The cheapest market that has the whole list, if any does.
  MarketComparison? get bestMarket {
    for (final m in markets) {
      if (m.market.id == bestSingleMarketId) return m;
    }
    return null;
  }

  /// Items no market sells at all.
  List<ListItemSummary> get unavailableItems => items.where((i) => !i.available).toList();

  int get completeCount => markets.where((m) => m.isComplete).length;
}

List<T> _list<T>(Object? raw, T Function(Map<String, dynamic>) build) {
  return (raw as List<dynamic>? ?? const []).map((e) => build(e as Map<String, dynamic>)).toList();
}
