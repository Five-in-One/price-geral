import '../models/models.dart';

/// The API returns one row per market listing, so the same product sold at two
/// markets shows up twice. Groups the listings of one catalog item into a
/// single row: the cheapest listing, tagged with how many markets sell it.
///
/// Listings that aren't in the catalog (no [Product.catalogId]) stay as they
/// are. The order of first appearance is kept, so a server-side sort (by name
/// or by price) still reads correctly.
List<Product> groupByCatalog(List<Product> listings) {
  final result = <Product?>[];
  final indexOf = <String, int>{};
  final markets = <String, Set<String>>{};

  for (final p in listings) {
    final id = p.catalogId;
    if (id == null) {
      result.add(p);
      continue;
    }

    final at = indexOf[id];
    markets.putIfAbsent(id, () => <String>{}).add(p.defaultMarketName);
    if (at == null) {
      indexOf[id] = result.length;
      result.add(p);
    } else if (p.defaultPrice < result[at]!.defaultPrice) {
      result[at] = p;
    }
  }

  return [
    for (final p in result)
      if (p!.catalogId != null) p.copyWith(marketsCount: markets[p.catalogId]!.length) else p,
  ];
}
