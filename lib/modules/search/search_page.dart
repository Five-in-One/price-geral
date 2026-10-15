import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../core/models/models.dart';
import '../../core/models/product_search.dart';
import '../../core/services/api_service.dart';
import '../../core/services/app_state.dart';
import '../../core/components/location_feedback.dart';
import '../../core/components/ui.dart';
import '../../core/components/widgets.dart';
import '../../core/theme/app_theme.dart';
import '../../core/services/location_service.dart';
import 'search_filters_sheet.dart';

class SearchPage extends StatefulWidget {
  /// Lets the screen around it scroll the list back to the top (tapping the
  /// "Busca" tab again). Without one, the page uses its own.
  final ScrollController? scrollController;

  const SearchPage({super.key, this.scrollController});

  /// How far down the list must be before the "back to top" button can show.
  static const scrollToTopThreshold = 600.0;

  // Lists being animated to the top; the "Topo" button stays hidden meanwhile.
  static final Set<ScrollController> _animating = {};

  /// Scrolls [controller]'s list back to the top, if it is attached.
  static Future<void> scrollToTop(ScrollController controller) async {
    if (!controller.hasClients) return;
    _animating.add(controller);
    try {
      await controller.animateTo(0, duration: const Duration(milliseconds: 450), curve: Curves.easeOutCubic);
    } finally {
      _animating.remove(controller);
    }
  }

  @override
  State<SearchPage> createState() => _SearchPageState();
}

class _SearchPageState extends State<SearchPage> {
  ScrollController? _ownController;
  ScrollController get _scroll => widget.scrollController ?? (_ownController ??= ScrollController());

  // The "back to top" button shows while the person scrolls up from far down
  // the list, and hides again when they scroll down or get near the top.
  // Decided from how the position moves, not from userScrollDirection: on
  // Android the fling that follows an upward drag reports "reverse".
  bool _showToTop = false;
  double _lastPixels = 0;
  // How far the list has gone up since it last went down. A few pixels are
  // not enough: the position also shifts when the content height changes.
  double _upDistance = 0;
  static const _upToShow = 24.0;

  @override
  void initState() {
    super.initState();
    _scroll.addListener(_onScroll);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<AppState>().refreshSearchResults();
    });
  }

  @override
  void didUpdateWidget(SearchPage oldWidget) {
    super.didUpdateWidget(oldWidget);
    final old = oldWidget.scrollController ?? _ownController;
    if (old != _scroll) {
      old?.removeListener(_onScroll);
      _scroll.addListener(_onScroll);
    }
  }

  @override
  void dispose() {
    _scroll.removeListener(_onScroll);
    _ownController?.dispose();
    super.dispose();
  }

  void _onScroll() {
    final pixels = _scroll.position.pixels;
    final delta = pixels - _lastPixels;
    _lastPixels = pixels;
    _upDistance = delta < 0 ? _upDistance - delta : 0;

    bool show = _showToTop;
    if (pixels < SearchPage.scrollToTopThreshold / 2 || SearchPage._animating.contains(_scroll)) {
      show = false; // near the top, or already on its way there
    } else if (_upDistance >= _upToShow && pixels > SearchPage.scrollToTopThreshold) {
      show = true; // going up, far from the top
    } else if (delta > 0) {
      show = false; // going down
    }
    if (show != _showToTop) setState(() => _showToTop = show);
  }

  Future<void> _backToTop() async {
    setState(() => _showToTop = false);
    await SearchPage.scrollToTop(_scroll);
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

  Future<void> _pickSort(BuildContext context) async {
    final appState = context.read<AppState>();
    final picked = await showModalBottomSheet<ProductSort>(
      context: context,
      showDragHandle: true,
      builder: (context) => SafeArea(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Padding(
              padding: EdgeInsets.fromLTRB(20, 0, 20, 8),
              child: Text('Ordenar por', style: TextStyle(fontSize: 16, fontWeight: FontWeight.w800)),
            ),
            for (final sort in ProductSort.values)
              ListTile(
                title: Text(sort.label),
                trailing: sort == appState.sort ? const Icon(Icons.check, color: AppColors.brand) : null,
                onTap: () => Navigator.pop(context, sort),
              ),
          ],
        ),
      ),
    );
    if (picked != null) appState.setSort(picked);
  }

  Future<void> _openFilters(BuildContext context) async {
    final appState = context.read<AppState>();
    final picked = await showModalBottomSheet<SearchFilters>(
      context: context,
      isScrollControlled: true,
      showDragHandle: true,
      builder: (context) => SearchFiltersSheet(
        initial: appState.searchFilters,
        markets: appState.markets,
        hasLocation: appState.userLocation != null,
        radiusKm: appState.radiusKm,
      ),
    );
    if (picked != null) appState.setSearchFilters(picked);
  }

  /// "40 de 780 produtos · página 2 de 39"; just the count when it all fits
  /// in one page.
  String _pagingText(AppState appState) {
    final total = appState.searchTotal;
    if (total == 1) return '1 produto encontrado';
    if (appState.searchPageCount <= 1) return '$total produtos encontrados';
    return '${appState.searchResults.length} de $total produtos · '
        'página ${appState.searchPagesLoaded} de ${appState.searchPageCount}';
  }

  @override
  Widget build(BuildContext context) {
    final appState = context.watch<AppState>();
    final selectedCategory = appState.selectedCategory;
    final products = appState.searchResults;
    final isLoading = appState.isLoadingProducts;
    final error = appState.productsError;

    // "Tudo" plus every category the server has for this search, with counts.
    // The selected one stays even when the current search empties it.
    final categories = <({String value, String label})>[
      (value: 'Tudo', label: 'Tudo'),
      for (final c in appState.categories) (value: c.category, label: '${c.label} (${c.count})'),
      if (selectedCategory != 'Tudo' &&
          !appState.categories.any((c) => c.category.toUpperCase() == selectedCategory.toUpperCase()))
        (value: selectedCategory, label: CategoryCount(selectedCategory, 0).label),
    ];

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
      results = SliverFillRemaining(
        hasScrollBody: false,
        child: EmptyState(
          icon: Icons.search_off_rounded,
          title: 'Nenhum produto encontrado.',
          message: appState.searchFilters.isEmpty
              ? 'Tente outro nome ou outra categoria.'
              : 'Tente outro nome, outra categoria ou menos filtros.',
        ),
      );
    } else {
      results = SliverPadding(
        padding: const EdgeInsets.fromLTRB(20, 0, 20, 16),
        sliver: SliverList.builder(
          // One extra row at the end: the next page loading, its error, or
          // the end of the list.
          itemCount: products.length + 1,
          itemBuilder: (context, index) {
            if (index < products.length) return productCard(products[index]);
            return _ListFooter(appState: appState);
          },
        ),
      );
    }

    return Scaffold(
      body: SafeArea(
        child: Stack(
          children: [
            CustomScrollView(
              controller: _scroll,
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
                            final isSelected = selectedCategory.toLowerCase() == cat.value.toLowerCase();

                            return Padding(
                              padding: const EdgeInsets.only(right: 8.0),
                              child: GestureDetector(
                                onTap: () => appState.setSelectedCategory(cat.value),
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
                                      if (cat.value != 'Tudo') ...[
                                        Icon(
                                          _getCategoryIcon(cat.value),
                                          size: 15,
                                          color: isSelected ? Colors.white : AppColors.gray,
                                        ),
                                        const SizedBox(width: 6),
                                      ],
                                      Text(
                                        cat.label,
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

                      // Sort, filters and how many products the search found
                      Row(
                        children: [
                          // Only the sort label gives way (ellipsis) when space runs out.
                          Flexible(
                            child: _Pill(
                              key: const ValueKey('search-sort'),
                              icon: Icons.swap_vert,
                              text: appState.sort.label,
                              active: true,
                              shrink: true,
                              onTap: () => _pickSort(context),
                            ),
                          ),
                          const SizedBox(width: 8),
                          _Pill(
                            key: const ValueKey('search-filters'),
                            icon: Icons.tune_outlined,
                            text: appState.searchFilters.isEmpty
                                ? 'Filtros'
                                : 'Filtros (${appState.searchFilters.activeCount})',
                            active: !appState.searchFilters.isEmpty,
                            onTap: () => _openFilters(context),
                          ),
                        ],
                      ),
                      const SizedBox(height: 10),
                      SizedBox(
                        height: 16,
                        child: Align(
                          alignment: Alignment.centerLeft,
                          child: isLoading
                              ? const SizedBox(width: 14, height: 14, child: CircularProgressIndicator(strokeWidth: 2))
                              : Text(
                                  error != null ? '' : _pagingText(appState),
                                  style: const TextStyle(
                                    fontSize: 11,
                                    color: AppColors.grayLight,
                                    fontWeight: FontWeight.w600,
                                  ),
                                ),
                        ),
                      ),
                      const SizedBox(height: 12),
                    ],
                  ),
                ),

                // Product list
                results,
              ],
            ),
            // Back to the top: small, centered, shown while scrolling up.
            Positioned(
              top: 10,
              left: 0,
              right: 0,
              child: Center(
                child: IgnorePointer(
                  ignoring: !_showToTop,
                  child: AnimatedSlide(
                    offset: _showToTop ? Offset.zero : const Offset(0, -1.5),
                    duration: const Duration(milliseconds: 200),
                    curve: Curves.easeOut,
                    child: AnimatedOpacity(
                      opacity: _showToTop ? 1 : 0,
                      duration: const Duration(milliseconds: 200),
                      child: _ToTopButton(onTap: _backToTop),
                    ),
                  ),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

/// A rounded pill button in the sort/filters row.
class _Pill extends StatelessWidget {
  final IconData icon;
  final String text;
  final bool active;
  final VoidCallback onTap;

  /// Lets the label shrink with an ellipsis; only inside a Flexible.
  final bool shrink;

  const _Pill({
    super.key,
    required this.icon,
    required this.text,
    required this.active,
    required this.onTap,
    this.shrink = false,
  });

  @override
  Widget build(BuildContext context) {
    final color = active ? AppColors.brand : AppColors.gray;
    final label = Text(
      text,
      maxLines: 1,
      overflow: TextOverflow.ellipsis,
      style: TextStyle(fontSize: 12, fontWeight: FontWeight.w700, color: color),
    );
    return Material(
      color: active ? AppColors.accentBg : Colors.white,
      shape: StadiumBorder(side: BorderSide(color: active ? AppColors.brand : AppColors.border)),
      clipBehavior: Clip.antiAlias,
      child: InkWell(
        onTap: onTap,
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(icon, size: 14, color: color),
              const SizedBox(width: 5),
              if (shrink) Flexible(child: label) else label,
              const SizedBox(width: 2),
              Icon(Icons.keyboard_arrow_down, size: 14, color: color),
            ],
          ),
        ),
      ),
    );
  }
}

/// The row under the last product: the next page loading, a retry when it
/// failed, or the end of the results.
class _ListFooter extends StatelessWidget {
  final AppState appState;

  const _ListFooter({required this.appState});

  @override
  Widget build(BuildContext context) {
    final pages = appState.searchPageCount;
    if (appState.isLoadingMoreProducts) {
      return Padding(
        padding: const EdgeInsets.symmetric(vertical: 16),
        child: Row(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            const SizedBox(width: 18, height: 18, child: CircularProgressIndicator(strokeWidth: 2.5)),
            const SizedBox(width: 10),
            Text(
              'Carregando página ${appState.searchPagesLoaded + 1} de $pages…',
              style: const TextStyle(fontSize: 12, color: AppColors.grayLight),
            ),
          ],
        ),
      );
    }
    if (appState.loadMoreError != null) {
      return Padding(
        padding: const EdgeInsets.symmetric(vertical: 8),
        child: Center(
          child: TextButton.icon(
            onPressed: appState.loadMoreSearchResults,
            icon: const Icon(Icons.refresh),
            label: const Text('Não deu para carregar mais. Tentar de novo'),
          ),
        ),
      );
    }
    if (appState.searchHasMore) {
      // The next page is only asked for when the person taps this.
      final next = (appState.searchTotal - appState.searchResults.length).clamp(0, ApiService.searchPageSize);
      return Padding(
        padding: const EdgeInsets.only(top: 4, bottom: 8),
        child: Column(
          children: [
            SizedBox(
              width: double.infinity,
              child: OutlinedButton.icon(
                key: const ValueKey('load-more'),
                onPressed: appState.loadMoreSearchResults,
                icon: const Icon(Icons.expand_more),
                label: Text('Carregar mais $next'),
              ),
            ),
            const SizedBox(height: 6),
            Text(
              '${appState.searchResults.length} de ${appState.searchTotal} produtos · '
              'página ${appState.searchPagesLoaded} de $pages',
              style: const TextStyle(fontSize: 11, color: AppColors.grayLight),
            ),
          ],
        ),
      );
    }
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 16),
      child: Center(
        child: Text(
          pages > 1 ? 'Fim dos resultados · $pages páginas' : 'Fim dos resultados',
          style: const TextStyle(fontSize: 11, color: AppColors.grayLight),
        ),
      ),
    );
  }
}

/// The small floating "Topo" pill that scrolls the results back up.
class _ToTopButton extends StatelessWidget {
  final VoidCallback onTap;

  const _ToTopButton({required this.onTap});

  @override
  Widget build(BuildContext context) {
    return Semantics(
      button: true,
      label: 'Voltar ao topo',
      child: Material(
        color: AppColors.brand,
        shape: const StadiumBorder(),
        elevation: 4,
        shadowColor: AppColors.brand.withValues(alpha: 0.4),
        clipBehavior: Clip.antiAlias,
        child: InkWell(
          key: const ValueKey('scroll-to-top'),
          onTap: onTap,
          child: const Padding(
            padding: EdgeInsets.fromLTRB(12, 8, 14, 8),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Icon(Icons.arrow_upward_rounded, size: 16, color: Colors.white),
                SizedBox(width: 4),
                Text(
                  'Topo',
                  style: TextStyle(fontSize: 12, fontWeight: FontWeight.w800, color: Colors.white),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
