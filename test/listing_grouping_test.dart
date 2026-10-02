import 'package:flutter_test/flutter_test.dart';
import 'package:price/core/models/models.dart';
import 'package:price/core/services/listing_grouping.dart';

import 'comparison_test_helpers.dart';

void main() {
  test('one row per catalog item: the cheapest listing, tagged with the market count', () {
    final grouped = groupByCatalog([
      product('a1', catalogId: 'nescau', price: 2.79, market: 'Carvalho'),
      product('b1', catalogId: 'heinz', price: 14.39, market: 'Atacadão'),
      product('a2', catalogId: 'nescau', price: 2.29, market: 'Atacadão'),
    ]);

    expect(grouped.map((p) => p.id), ['a2', 'b1']);
    expect(grouped[0].defaultMarketName, 'Atacadão');
    expect(grouped[0].defaultPrice, 2.29);
    expect(grouped[0].marketsCount, 2);
    expect(grouped[1].marketsCount, 1);
  });

  test('keeps the position of the first listing, so the server sort still reads right', () {
    final grouped = groupByCatalog([
      product('x', catalogId: 'x', price: 1),
      product('y1', catalogId: 'y', price: 9, market: 'Carvalho'),
      product('z', catalogId: 'z', price: 3),
      product('y2', catalogId: 'y', price: 2, market: 'Atacadão'),
    ]);

    expect(grouped.map((p) => p.catalogId), ['x', 'y', 'z']);
    expect(grouped[1].id, 'y2'); // cheapest, but still second
  });

  test('two listings of one item at the same market count as one market', () {
    final grouped = groupByCatalog([
      product('a', catalogId: 'k', price: 5, market: 'Atacadão'),
      product('b', catalogId: 'k', price: 6, market: 'Atacadão'),
    ]);

    expect(grouped, hasLength(1));
    expect(grouped.single.marketsCount, 1);
  });

  test('listings outside the catalog are left alone', () {
    final grouped = groupByCatalog([
      product('no-catalog-1', name: 'Mesmo nome'),
      product('no-catalog-2', name: 'Mesmo nome'),
    ]);

    expect(grouped, hasLength(2), reason: 'without a catalog id there is no way to know they are the same');
    expect(grouped.every((p) => p.marketsCount == 1), isTrue);
  });

  test('a product survives being saved on the device and read back', () {
    final original = Product(
      id: 'p',
      name: 'Café Melitta 500g',
      unit: 'un',
      category: 'GRÃOS',
      defaultMarketName: 'Atacadão',
      defaultPrice: 28.9,
      catalogId: 'cafe',
      brand: 'Melitta',
      sizeLabel: '500 g',
      unitPrice: 57.8,
      unitPriceUnit: 'kg',
    );

    final restored = Product.fromJson(original.toJson());

    expect(restored.catalogId, 'cafe');
    expect(restored.brand, 'Melitta');
    expect(restored.sizeLabel, '500 g');
    expect(restored.unitPrice, 57.8);
    expect(restored.defaultPrice, 28.9);
  });

  test('an empty catalog_id from the API means "not in the catalog"', () {
    final p = Product.fromJson({
      'id': 'p',
      'name': 'X',
      'unit': 'un',
      'category': 'OUTROS',
      'default_market_name': 'M',
      'default_price': 1,
      'catalog_id': '',
    });
    expect(p.catalogId, isNull);
  });
}
