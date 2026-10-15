import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../core/models/models.dart';
import '../../core/services/app_state.dart';
import '../../core/components/location_feedback.dart';
import '../../core/components/ui.dart';
import '../../core/components/widgets.dart';
import '../../core/theme/app_theme.dart';
import '../../core/services/location_service.dart';

class SearchPage extends StatefulWidget {
  const SearchPage({super.key});

  @override
  State<SearchPage> createState() => _SearchPageState();
}

class _SearchPageState extends State<SearchPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<AppState>().refreshSearchResults();
    });
  }

  Future<void> _updateLocation(BuildContext context) async {
    final appState = context.read<AppState>();
    final messenger = ScaffoldMessenger.of(context);
    try {
      await appState.updateLocation();
      messenger.showSnackBar(const SnackBar(content: Text('Localização atualizada.'), duration: Duration(seconds: 2)));
    } on LocationException catch (e) {
      showLocationError(messenger, appState, e);
    }
  }

  /// "· 500 g · R$ 57,80/kg" when the package size is known, else "/un".
  String _sizeText(Product product) {
    if (product.sizeLabel.isEmpty) return '/${product.unit}';
    final unitPrice = product.unitPrice;
    final perUnit = unitPrice == null ? '' : ' · ${formatBrl(unitPrice)}/${product.unitPriceUnit}';
    return '· ${product.sizeLabel}$perUnit';
  }

  IconData _getCategoryIcon(String category) {
    switch (category.toUpperCase()) {
      case 'HORTIFRUTI':
        return Icons.eco_outlined;
      case 'GRÃOS':
        return Icons.grain_outlined;
      case 'LATICÍNIOS':
        return Icons.water_drop_outlined;
      case 'CARNES':
        return Icons.kebab_dining_outlined;
      default:
        return Icons.shopping_basket_outlined;
    }
  }

  @override
  Widget build(BuildContext context) {
    final appState = context.watch<AppState>();
    final selectedCategory = appState.selectedCategory;
    final selectedFilter = appState.selectedFilter;
    final products = appState.searchResults;
    final isLoading = appState.isLoadingProducts;
    final error = appState.productsError;

    final categories = ['Tudo', 'Hortifruti', 'Grãos', 'Laticínios', 'Carnes'];
    final filters = ['Preço em', 'Menor preço', 'do Bairro', 'Supermercados'];

    Widget productCard(Product product) {
      final int quantity = appState.getProductQuantity(product);
      return Padding(
        padding: const EdgeInsets.only(bottom: 12),
        child: Container(
          padding: const EdgeInsets.all(14),
          decoration: BoxDecoration(
            color: Colors.white,
            borderRadius: BorderRadius.circular(20),
            border: Border.all(color: AppColors.border),
          ),
          child: Row(
            children: [
              // Category icon in a soft blue tile
              IconTile(_getCategoryIcon(product.category), size: 46),
              const SizedBox(width: 14),

              // Details
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      product.name,
                      style: const TextStyle(fontSize: 14, fontWeight: FontWeight.w800, color: AppColors.ink),
                    ),
                    const SizedBox(height: 6),
                    // Price first, market beside it when it fits or on the next line
                    // (wrapping its own text) on narrow screens / long market names.
                    Wrap(
                      crossAxisAlignment: WrapCrossAlignment.center,
                      spacing: 8,
                      runSpacing: 4,
                      children: [
                        Row(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            Text(
                              'R\$ ${product.defaultPrice.toStringAsFixed(2).replaceAll('.', ',')} ',
                              style: const TextStyle(fontSize: 15, fontWeight: FontWeight.w900, color: AppColors.brand),
                            ),
                            Text(
                              _sizeText(product),
                              style: const TextStyle(
                                fontSize: 11,
                                color: AppColors.grayLight,
                                fontWeight: FontWeight.w500,
                              ),
                            ),
                          ],
                        ),
                        Row(
                          mainAxisSize: MainAxisSize.min,
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            const Padding(
                              padding: EdgeInsets.only(top: 1),
                              child: Icon(Icons.storefront_outlined, size: 12, color: AppColors.grayLight),
                            ),
                            const SizedBox(width: 4),
                            Flexible(
                              child: Text(
                                product.marketsCount > 1
                                    ? '@ ${product.defaultMarketName} · em ${product.marketsCount} mercados'
                                    : '@ ${product.defaultMarketName}',
                                style: const TextStyle(fontSize: 11, color: AppColors.grayLight),
                              ),
                            ),
                          ],
                        ),
                      ],
                    ),
                  ],
                ),
              ),
              const SizedBox(width: 8),

              // Add button or Quantitative Selector
              QuantitySelector(
                quantity: quantity,
                onIncrement: () => appState.updateQuantity(product, 1),
                onDecrement: () => appState.updateQuantity(product, -1),
              ),
            ],
          ),
        ),
      );
    }

    final Widget results;
    if (error != null) {
      results = SliverFillRemaining(
        hasScrollBody: false,
        child: ErrorRetry(message: error, onRetry: () => appState.refreshSearchResults()),
      );
    } else if (isLoading && products.isEmpty) {
      results = const SliverFillRemaining(hasScrollBody: false, child: Center(child: CircularProgressIndicator()));
    } else if (products.isEmpty) {
      results = const SliverFillRemaining(
        hasScrollBody: false,
        child: EmptyState(
          icon: Icons.search_off_rounded,
          title: 'Nenhum produto encontrado.',
          message: 'Tente outro nome ou outra categoria.',
        ),
      );
    } else {
      results = SliverPadding(
        padding: const EdgeInsets.fromLTRB(20, 0, 20, 16),
        sliver: SliverList.builder(
          itemCount: products.length,
          itemBuilder: (context, index) => productCard(products[index]),
        ),
      );
    }

    return Scaffold(
      body: SafeArea(
        child: CustomScrollView(
          slivers: [
            SliverPadding(
              padding: const EdgeInsets.fromLTRB(20, 10, 20, 0),
              sliver: SliverList.list(
                children: [
                  // Top location pill: tap to read the GPS again
                  Align(
                    alignment: Alignment.centerLeft,
                    child: Semantics(
                      button: true,
                      label: 'Atualizar minha localização',
                      child: Material(
                        color: Colors.white,
                        shape: const StadiumBorder(side: BorderSide(color: AppColors.border)),
                        clipBehavior: Clip.antiAlias,
                        child: InkWell(
                          key: const ValueKey('update-location'),
                          onTap: appState.isUpdatingLocation ? null : () => _updateLocation(context),
                          child: Padding(
                            padding: const EdgeInsets.fromLTRB(7, 8, 14, 8),
                            child: Row(
                              mainAxisSize: MainAxisSize.min,
                              children: [
                                Container(
                                  width: 26,
                                  height: 26,
                                  decoration: const BoxDecoration(color: AppColors.brand, shape: BoxShape.circle),
                                  child: const Icon(Icons.location_on, color: Colors.white, size: 15),
                                ),
                                const SizedBox(width: 8),
                                Flexible(
                                  child: Text(
                                    appState.currentAddress ?? 'Selecionar local...',
                                    overflow: TextOverflow.ellipsis,
                                    style: const TextStyle(
                                      fontSize: 13,
                                      color: AppColors.ink,
                                      fontWeight: FontWeight.w700,
                                    ),
                                  ),
                                ),
                                const SizedBox(width: 8),
                                appState.isUpdatingLocation
                                    ? const SizedBox(
                                        width: 14,
                                        height: 14,
                                        child: CircularProgressIndicator(strokeWidth: 2),
                                      )
                                    : const Icon(Icons.refresh, color: AppColors.grayLight, size: 16),
                              ],
                            ),
                          ),
                        ),
                      ),
                    ),
                  ),
                  const SizedBox(height: 14),

                  HeroBanner(
                    eyebrow: 'Busca',
                    title: 'O que você',
                    highlight: 'precisa?',
                    emoji: '🥦',
                    children: [
                      // Search Bar
                      Container(
                        decoration: BoxDecoration(
                          color: Colors.white,
                          borderRadius: BorderRadius.circular(999),
                          boxShadow: AppColors.softShadow,
                        ),
                        child: TextField(
                          onChanged: (val) => appState.setSearchQuery(val),
                          decoration: const InputDecoration(
                            hintText: 'Buscar produto...',
                            hintStyle: TextStyle(color: AppColors.grayLight, fontSize: 14),
                            prefixIcon: Icon(Icons.search, color: AppColors.brand, size: 22),
                            filled: false,
                            border: InputBorder.none,
                            enabledBorder: InputBorder.none,
                            focusedBorder: InputBorder.none,
                            contentPadding: EdgeInsets.symmetric(vertical: 15),
                          ),
                          style: const TextStyle(fontSize: 14, color: AppColors.ink),
                        ),
                      ),
                    ],
                  ),
                  const SizedBox(height: 16),

                  // Horizontal Category List
                  SizedBox(
                    height: 40,
                    child: ListView.builder(
                      scrollDirection: Axis.horizontal,
                      itemCount: categories.length,
                      itemBuilder: (context, index) {
                        final cat = categories[index];
                        final isSelected = selectedCategory.toLowerCase() == cat.toLowerCase();

                        return Padding(
                          padding: const EdgeInsets.only(right: 8.0),
                          child: GestureDetector(
                            onTap: () => appState.setSelectedCategory(cat),
                            child: AnimatedContainer(
                              duration: const Duration(milliseconds: 150),
                              padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 9),
                              decoration: BoxDecoration(
                                color: isSelected ? AppColors.brand : Colors.white,
                                borderRadius: BorderRadius.circular(999),
                                border: Border.all(color: isSelected ? AppColors.brand : AppColors.border),
                              ),
                              child: Row(
                                children: [
                                  if (cat != 'Tudo') ...[
                                    Icon(
                                      _getCategoryIcon(cat),
                                      size: 15,
                                      color: isSelected ? Colors.white : AppColors.gray,
                                    ),
                                    const SizedBox(width: 6),
                                  ],
                                  Text(
                                    cat,
                                    style: TextStyle(
                                      fontSize: 13,
                                      fontWeight: FontWeight.w700,
                                      color: isSelected ? Colors.white : AppColors.gray,
                                    ),
                                  ),
                                ],
                              ),
                            ),
                          ),
                        );
                      },
                    ),
                  ),
                  const SizedBox(height: 10),

                  // Filter pills row
                  SizedBox(
                    height: 32,
                    child: ListView.builder(
                      scrollDirection: Axis.horizontal,
                      itemCount: filters.length,
                      itemBuilder: (context, index) {
                        final filter = filters[index];
                        final isMenorPreco = filter == 'Menor preço';
                        final isSelected =
                            selectedFilter == filter || (isMenorPreco && selectedFilter == 'Menor preço');
                        final active = isSelected && isMenorPreco;

                        return Padding(
                          padding: const EdgeInsets.only(right: 8.0),
                          child: GestureDetector(
                            onTap: () => appState.setSelectedFilter(filter),
                            child: Container(
                              padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
                              decoration: BoxDecoration(
                                color: active ? AppColors.accentBg : Colors.transparent,
                                borderRadius: BorderRadius.circular(999),
                                border: Border.all(color: active ? AppColors.brand : AppColors.border),
                              ),
                              child: Row(
                                mainAxisSize: MainAxisSize.min,
                                children: [
                                  if (filter == 'Preço em') ...[
                                    const Icon(Icons.tune_outlined, size: 12, color: AppColors.grayLight),
                                    const SizedBox(width: 4),
                                  ],
                                  Text(
                                    filter,
                                    style: TextStyle(
                                      fontSize: 11,
                                      fontWeight: FontWeight.w700,
                                      color: active ? AppColors.brand : AppColors.gray,
                                    ),
                                  ),
                                  if (isMenorPreco || filter == 'Preço em') ...[
                                    const SizedBox(width: 4),
                                    Icon(
                                      Icons.keyboard_arrow_down,
                                      size: 12,
                                      color: active ? AppColors.brand : AppColors.grayLight,
                                    ),
                                  ],
                                ],
                              ),
                            ),
                          ),
                        );
                      },
                    ),
                  ),
                  const SizedBox(height: 18),
                ],
              ),
            ),

            // Product list
            results,
          ],
        ),
      ),
    );
  }
}
