import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../core/models/list_comparison.dart';
import '../../core/models/models.dart';
import '../../core/services/app_state.dart';
import '../../core/components/market_address.dart';
import '../../core/components/ui.dart';
import '../../core/components/widgets.dart';
import '../../core/theme/app_theme.dart';

const _blue = AppColors.brand;
const _ink = AppColors.ink;
const _gray = AppColors.grayLight;
const _red = Color(0xFFDC2626);
const _amber = Color(0xFFB45309);

class ComparisonPage extends StatefulWidget {
  const ComparisonPage({super.key});

  @override
  State<ComparisonPage> createState() => _ComparisonPageState();
}

class _ComparisonPageState extends State<ComparisonPage> {
  // Markets whose item breakdown is open
  final Set<String> _expandedMarkets = {};
  bool _combinationExpanded = false;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<AppState>().refreshComparison();
    });
  }

  void _toggleMarket(String marketId) {
    setState(() {
      if (!_expandedMarkets.remove(marketId)) _expandedMarkets.add(marketId);
    });
  }

  @override
  Widget build(BuildContext context) {
    final appState = context.watch<AppState>();

    if (appState.totalItemsCount == 0) {
      return Scaffold(
        body: SafeArea(
          child: EmptyState(
            icon: Icons.balance_outlined,
            title: 'Adicione itens para ver a comparação.',
            message: 'Monte sua lista na Busca e veja quanto ela custa em cada mercado perto de você.',
            action: ElevatedButton.icon(
              onPressed: () => appState.setTabIndex(0),
              icon: const Icon(Icons.search, size: 18),
              label: const Text('Buscar produtos'),
            ),
          ),
        ),
      );
    }

    final comparison = appState.comparison;

    if (comparison == null) {
      return Scaffold(
        body: SafeArea(
          child: appState.comparisonError != null
              ? ErrorRetry(message: appState.comparisonError!, onRetry: appState.refreshComparison)
              : const Center(child: CircularProgressIndicator(color: _blue)),
        ),
      );
    }

    final combination = comparison.bestCombination;
    final showCombination = combination != null && combination.stops.length > 1;

    return Scaffold(
      body: SafeArea(
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Expanded(
              child: ListView(
                padding: const EdgeInsets.fromLTRB(20, 8, 20, 0),
                children: [
                  // Back to the list tab, and a spinner while the comparison reloads
                  Row(
                    children: [
                      Material(
                        color: Colors.white,
                        shape: const CircleBorder(side: BorderSide(color: AppColors.border)),
                        child: IconButton(
                          icon: const Icon(Icons.arrow_back_ios_new, color: _ink, size: 16),
                          onPressed: () => appState.setTabIndex(1), // Back to Cart Page
                        ),
                      ),
                      const Spacer(),
                      if (appState.isLoadingComparison)
                        const SizedBox(
                          width: 18,
                          height: 18,
                          child: CircularProgressIndicator(strokeWidth: 2, color: _blue),
                        ),
                    ],
                  ),
                  const SizedBox(height: 12),
                  HeroBanner(
                    eyebrow: 'Comparar',
                    title: 'Comparação',
                    highlight: 'de preços',
                    emoji: '⚖️',
                    children: [
                      Wrap(
                        spacing: 8,
                        runSpacing: 8,
                        children: [
                          StatChip('${appState.totalItemsCount} itens'),
                          StatChip('${comparison.markets.length} mercados', dot: AppColors.ok),
                        ],
                      ),
                    ],
                  ),
                  const SizedBox(height: 16),
                  _EquivalentsToggle(
                    value: appState.allowEquivalents,
                    onChanged: appState.setAllowEquivalents,
                  ),
                  const SizedBox(height: 14),

                  _SummaryCard(comparison: comparison),
                  const SizedBox(height: 12),

                  if (comparison.unavailableItems.isNotEmpty) ...[
                    _UnavailableNotice(items: comparison.unavailableItems),
                    const SizedBox(height: 12),
                  ],

                  if (showCombination) ...[
                    _CombinationCard(
                      combination: combination,
                      bestMarket: comparison.bestMarket,
                      expanded: _combinationExpanded,
                      onToggle: () => setState(() => _combinationExpanded = !_combinationExpanded),
                    ),
                    const SizedBox(height: 12),
                  ],

                  const SizedBox(height: 12),
                  const Row(
                    mainAxisAlignment: MainAxisAlignment.spaceBetween,
                    children: [
                      Text(
                        'TODOS OS MERCADOS',
                        style: TextStyle(fontSize: 11, fontWeight: FontWeight.bold, color: _gray, letterSpacing: 0.5),
                      ),
                      SizedBox(width: 8),
                      Flexible(
                        child: Text(
                          'completos primeiro',
                          style: TextStyle(fontSize: 11, color: _gray),
                          overflow: TextOverflow.ellipsis,
                        ),
                      ),
                    ],
                  ),
                  const SizedBox(height: 10),

                  for (var i = 0; i < comparison.markets.length; i++)
                    _MarketRow(
                      rank: i + 1,
                      result: comparison.markets[i],
                      isBest: comparison.markets[i].market.id == comparison.bestSingleMarketId,
                      expanded: _expandedMarkets.contains(comparison.markets[i].market.id),
                      onTap: () => _toggleMarket(comparison.markets[i].market.id),
                    ),
                  const SizedBox(height: 20),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

/// "Aceitar outras marcas": lets a market complete the list with another
/// brand of the same product when it lacks the one asked for.
class _EquivalentsToggle extends StatelessWidget {
  final bool value;
  final ValueChanged<bool> onChanged;

  const _EquivalentsToggle({required this.value, required this.onChanged});

  @override
  Widget build(BuildContext context) {
    return AppCard(
      padding: const EdgeInsets.fromLTRB(14, 12, 8, 12),
      child: Row(
        children: [
          const IconTile(Icons.swap_horiz, size: 38),
          const SizedBox(width: 12),
          const Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('Aceitar outras marcas', style: TextStyle(fontSize: 13, fontWeight: FontWeight.bold, color: _ink)),
                SizedBox(height: 2),
                Text(
                  'Só para arroz, feijão, açúcar e similares. A marca da sua lista tem prioridade.',
                  style: TextStyle(fontSize: 11, color: _gray),
                ),
              ],
            ),
          ),
          Switch(value: value, activeThumbColor: _blue, onChanged: onChanged),
        ],
      ),
    );
  }
}

/// The headline: the cheapest market that has the whole list — or a warning
/// that none does.
class _SummaryCard extends StatelessWidget {
  final ListComparison comparison;

  const _SummaryCard({required this.comparison});

  @override
  Widget build(BuildContext context) {
    final best = comparison.bestMarket;

    if (best == null) {
      return Container(
        padding: const EdgeInsets.all(18),
        decoration: BoxDecoration(
          color: AppColors.warnBg,
          borderRadius: BorderRadius.circular(22),
          border: Border.all(color: const Color(0xFFFCD9A0)),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Row(
              children: [
                Icon(Icons.info_outline, color: _amber, size: 16),
                SizedBox(width: 6),
                Expanded(
                  child: Text(
                    'NENHUM MERCADO TEM TODOS OS ITENS',
                    style: TextStyle(fontSize: 11, fontWeight: FontWeight.bold, color: _amber, letterSpacing: 0.3),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 8),
            Text(
              comparison.allowEquivalents
                  ? 'Veja abaixo o que falta em cada mercado.'
                  : 'Veja abaixo o que falta em cada mercado, ou ative "Aceitar outras marcas" para completar a lista.',
              style: const TextStyle(fontSize: 12, color: Color(0xFF636366)),
            ),
          ],
        ),
      );
    }

    return Container(
      clipBehavior: Clip.antiAlias,
      decoration: BoxDecoration(
        gradient: const LinearGradient(
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
          colors: [AppColors.brandBright, AppColors.brand, AppColors.brandDark],
        ),
        borderRadius: BorderRadius.circular(24),
        boxShadow: [BoxShadow(color: AppColors.brand.withValues(alpha: 0.3), blurRadius: 24, offset: const Offset(0, 10))],
      ),
      child: Stack(
        children: [
          Positioned(
            right: -40,
            top: -50,
            child: Container(
              width: 150,
              height: 150,
              decoration: BoxDecoration(shape: BoxShape.circle, color: Colors.white.withValues(alpha: 0.08)),
            ),
          ),
          Positioned(
            left: -30,
            bottom: -60,
            child: Container(
              width: 120,
              height: 120,
              decoration: BoxDecoration(
                shape: BoxShape.circle,
                border: Border.all(color: Colors.white.withValues(alpha: 0.18), width: 2),
              ),
            ),
          ),
          Padding(
            padding: const EdgeInsets.all(18),
            child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              const Icon(Icons.stars, color: Colors.white, size: 14),
              const SizedBox(width: 4),
              Text(
                'MELHOR PREÇO',
                style: TextStyle(
                  fontSize: 11,
                  fontWeight: FontWeight.bold,
                  color: Colors.white.withValues(alpha: 0.9),
                  letterSpacing: 0.5,
                ),
              ),
            ],
          ),
          const SizedBox(height: 14),
          Row(
            children: [
              MarketAvatar(
                initials: best.market.initials,
                primaryColor: best.market.primaryColor,
                backgroundColor: Colors.white,
                size: 40,
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      best.market.name,
                      style: const TextStyle(color: Colors.white, fontSize: 15, fontWeight: FontWeight.bold),
                    ),
                    const SizedBox(height: 2),
                    Text(
                      '${formatBrl(best.totalCost)} no total'
                      '${best.distanceKm == null ? '' : ' · ${formatKm(best.distanceKm!)}'}',
                      style: TextStyle(color: Colors.white.withValues(alpha: 0.85), fontSize: 12),
                    ),
                    if (best.market.hasAddress) ...[
                      const SizedBox(height: 4),
                      MarketAddressLine(market: best.market, color: Colors.white, maxLines: 2),
                    ],
                    MarketAddressButton(market: best.market, distanceKm: best.distanceKm, color: Colors.white),
                  ],
                ),
              ),
            ],
          ),
          const SizedBox(height: 18),
          if (comparison.maxSavings > 0)
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              crossAxisAlignment: CrossAxisAlignment.end,
              children: [
                Flexible(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        formatBrl(comparison.maxSavings),
                        style: const TextStyle(color: Colors.white, fontSize: 32, fontWeight: FontWeight.w900, height: 1.0, letterSpacing: -0.8),
                      ),
                      const SizedBox(height: 4),
                      Text(
                        'de diferença entre o mais barato e o mais caro que têm a lista toda',
                        style: TextStyle(color: Colors.white.withValues(alpha: 0.9), fontSize: 11, fontWeight: FontWeight.w500),
                      ),
                    ],
                  ),
                ),
                const SizedBox(width: 10),
                Container(
                  padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 7),
                  decoration: BoxDecoration(
                    color: Colors.white.withValues(alpha: 0.2),
                    borderRadius: BorderRadius.circular(999),
                  ),
                  child: Text(
                    '-${comparison.maxSavingsPercent.round()}%',
                    style: const TextStyle(color: Colors.white, fontSize: 11, fontWeight: FontWeight.bold),
                  ),
                ),
              ],
            )
          else
            Text(
              'É o único mercado com todos os itens da lista.',
              style: TextStyle(color: Colors.white.withValues(alpha: 0.9), fontSize: 12, fontWeight: FontWeight.w500),
            ),
        ],
      ),
          ),
        ],
      ),
    );
  }
}

class _UnavailableNotice extends StatelessWidget {
  final List<ListItemSummary> items;

  const _UnavailableNotice({required this.items});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: AppColors.redBg,
        borderRadius: BorderRadius.circular(20),
        border: Border.all(color: const Color(0xFFFBCACA)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            items.length == 1 ? 'Nenhum mercado tem este item:' : 'Nenhum mercado tem estes itens:',
            style: const TextStyle(fontSize: 12, fontWeight: FontWeight.bold, color: _red),
          ),
          const SizedBox(height: 4),
          for (final item in items)
            Text('• ${item.name}', style: const TextStyle(fontSize: 12, color: Color(0xFF636366))),
        ],
      ),
    );
  }
}

/// Buying each item where it is cheapest. The server doesn't weigh the trip,
/// so the stops are shown next to the saving and the person decides.
class _CombinationCard extends StatelessWidget {
  final BestCombination combination;
  final MarketComparison? bestMarket;
  final bool expanded;
  final VoidCallback onToggle;

  const _CombinationCard({
    required this.combination,
    required this.bestMarket,
    required this.expanded,
    required this.onToggle,
  });

  @override
  Widget build(BuildContext context) {
    final savings = combination.savingsVsBestSingle;
    final String detail;
    if (savings != null && bestMarket != null) {
      detail = savings > 0
          ? '${formatBrl(savings)} a menos do que comprar tudo em ${bestMarket!.market.name}.'
          : 'Sem economia em relação a ${bestMarket!.market.name}.';
    } else {
      detail = 'Nenhum mercado tem tudo: esta é a forma de comprar o que existe.';
    }

    return AppCard(
      padding: const EdgeInsets.all(14),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          GestureDetector(
            behavior: HitTestBehavior.opaque,
            onTap: onToggle,
            child: Row(
              children: [
                const IconTile(Icons.alt_route, size: 38),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        'Comprando em ${combination.stops.length} mercados: ${formatBrl(combination.totalCost)}',
                        style: const TextStyle(fontSize: 13, fontWeight: FontWeight.bold, color: _ink),
                      ),
                      const SizedBox(height: 2),
                      Text(detail, style: const TextStyle(fontSize: 11, color: Color(0xFF636366))),
                    ],
                  ),
                ),
                Icon(expanded ? Icons.keyboard_arrow_up : Icons.keyboard_arrow_down, size: 16, color: _gray),
              ],
            ),
          ),
          if (expanded) ...[
            const SizedBox(height: 10),
            for (final stop in combination.stops) ...[
              InkWell(
                onTap: () => showMarketDetails(context, stop.market, distanceKm: stop.distanceKm),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      children: [
                        Expanded(
                          child: Text(
                            stop.market.name,
                            style: const TextStyle(fontSize: 12, fontWeight: FontWeight.bold, color: _ink),
                            overflow: TextOverflow.ellipsis,
                          ),
                        ),
                        Text(
                          '${formatBrl(stop.subtotal)}${stop.distanceKm == null ? '' : ' · ${formatKm(stop.distanceKm!)}'}',
                          style: const TextStyle(fontSize: 11, color: _gray),
                        ),
                      ],
                    ),
                    MarketAddressLine(market: stop.market),
                  ],
                ),
              ),
              for (final line in stop.lines)
                Padding(
                  padding: const EdgeInsets.only(left: 8, top: 2),
                  child: Text(
                    '${line.quantity}x ${line.productName}',
                    style: const TextStyle(fontSize: 11, color: Color(0xFF636366)),
                  ),
                ),
              const SizedBox(height: 8),
            ],
            const Text(
              'O cálculo não considera o custo do deslocamento entre os mercados.',
              style: TextStyle(fontSize: 10, color: _gray, fontStyle: FontStyle.italic),
            ),
          ],
        ],
      ),
    );
  }
}

class _MarketRow extends StatelessWidget {
  final int rank;
  final MarketComparison result;
  final bool isBest;
  final bool expanded;
  final VoidCallback onTap;

  const _MarketRow({
    required this.rank,
    required this.result,
    required this.isBest,
    required this.expanded,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    final complete = result.isComplete;
    final market = result.market;

    return Column(
      children: [
        GestureDetector(
          behavior: HitTestBehavior.opaque,
          onTap: onTap,
          child: Padding(
            padding: const EdgeInsets.symmetric(vertical: 8.0),
            child: Container(
              padding: const EdgeInsets.all(14),
              decoration: BoxDecoration(
                color: isBest ? AppColors.accentBg : Colors.white,
                borderRadius: BorderRadius.circular(20),
                border: Border.all(
                  color: isBest ? AppColors.brand : AppColors.border,
                  width: isBest ? 2 : 1.0,
                ),
              ),
              child: Row(
                children: [
                  // Rank for markets with the whole list; a warning for the others
                  Container(
                    width: 24,
                    height: 24,
                    decoration: BoxDecoration(
                      color: isBest ? _blue : (complete ? AppColors.accentBg : AppColors.warnBg),
                      shape: BoxShape.circle,
                    ),
                    child: Center(
                      child: complete
                          ? Text(
                              '$rank',
                              style: TextStyle(
                                color: isBest ? Colors.white : AppColors.brand,
                                fontSize: 11,
                                fontWeight: FontWeight.bold,
                              ),
                            )
                          : const Icon(Icons.priority_high, size: 13, color: _amber),
                    ),
                  ),
                  const SizedBox(width: 8),
                  MarketAvatar(
                    initials: market.initials,
                    primaryColor: market.primaryColor,
                    backgroundColor: market.backgroundColor,
                    size: 38,
                  ),
                  const SizedBox(width: 10),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          market.name,
                          style: const TextStyle(fontSize: 14, fontWeight: FontWeight.w800, color: _ink),
                        ),
                        const SizedBox(height: 4),
                        Wrap(
                          spacing: 6,
                          runSpacing: 2,
                          crossAxisAlignment: WrapCrossAlignment.center,
                          children: [
                            Text(
                              complete
                                  ? '${result.totalItems} itens'
                                  : 'faltam ${result.missingCount} de ${result.totalItems} itens',
                              style: TextStyle(
                                fontSize: 11,
                                color: complete ? _gray : _amber,
                                fontWeight: complete ? FontWeight.normal : FontWeight.w600,
                              ),
                            ),
                            if (result.distanceKm != null)
                              Text('· ${formatKm(result.distanceKm!)}', style: const TextStyle(fontSize: 11, color: _gray)),
                            if (result.substitutesCount > 0)
                              _Chip(
                                text: result.substitutesCount == 1 ? '1 outra marca' : '${result.substitutesCount} outras marcas',
                                color: _amber,
                              ),
                            if (isBest) const _Chip(text: 'menor preço', color: _blue),
                          ],
                        ),
                        if (market.hasAddress) ...[
                          const SizedBox(height: 3),
                          MarketAddressLine(market: market, maxLines: 2),
                        ],
                      ],
                    ),
                  ),
                  Column(
                    crossAxisAlignment: CrossAxisAlignment.end,
                    children: [
                      Text(
                        formatBrl(result.totalCost),
                        style: TextStyle(
                          fontSize: 15,
                          fontWeight: FontWeight.w900,
                          color: complete ? _ink : _gray,
                        ),
                      ),
                      const SizedBox(height: 2),
                      if (!complete)
                        const Text('parcial', style: TextStyle(color: _amber, fontSize: 10, fontWeight: FontWeight.bold))
                      else if (isBest)
                        const Text('melhor total', style: TextStyle(color: _blue, fontSize: 10, fontWeight: FontWeight.bold))
                      else
                        Text(
                          '+ ${formatBrl(result.difference)}',
                          style: const TextStyle(color: _red, fontSize: 10, fontWeight: FontWeight.bold),
                        ),
                    ],
                  ),
                  const SizedBox(width: 6),
                  Icon(expanded ? Icons.keyboard_arrow_up : Icons.keyboard_arrow_down, size: 16, color: _gray),
                ],
              ),
            ),
          ),
        ),
        if (expanded) _Breakdown(result: result),
      ],
    );
  }
}

class _Chip extends StatelessWidget {
  final String text;
  final Color color;

  const _Chip({required this.text, required this.color});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2.5),
      decoration: BoxDecoration(color: color.withValues(alpha: 0.12), borderRadius: BorderRadius.circular(999)),
      child: Text(text, style: TextStyle(color: color, fontSize: 10, fontWeight: FontWeight.w800)),
    );
  }
}

/// Item-by-item prices at one market: what it charges, which lines no other
/// market beats, which use another brand, and what it doesn't sell.
class _Breakdown extends StatelessWidget {
  final MarketComparison result;

  const _Breakdown({required this.result});

  @override
  Widget build(BuildContext context) {
    return Container(
      margin: const EdgeInsets.only(left: 10, right: 10, bottom: 8),
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: AppColors.field,
        borderRadius: const BorderRadius.only(bottomLeft: Radius.circular(18), bottomRight: Radius.circular(18)),
        border: Border.all(color: AppColors.border),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Align(
            alignment: Alignment.centerLeft,
            child: MarketAddressButton(market: result.market, distanceKm: result.distanceKm),
          ),
          const SizedBox(height: 6),
          const Text(
            'Preços neste mercado:',
            style: TextStyle(fontSize: 11, fontWeight: FontWeight.bold, color: Color(0xFF636366)),
          ),
          const SizedBox(height: 6),
          for (final line in result.lines) _LineRow(line: line),
          for (final missing in result.missing)
            Padding(
              padding: const EdgeInsets.symmetric(vertical: 2.0),
              child: Row(
                children: [
                  const Icon(Icons.remove_circle_outline, size: 12, color: _red),
                  const SizedBox(width: 4),
                  Expanded(
                    child: Text(
                      'Não tem: ${missing.name}',
                      style: const TextStyle(fontSize: 12, color: _red),
                    ),
                  ),
                ],
              ),
            ),
        ],
      ),
    );
  }
}

class _LineRow extends StatelessWidget {
  final ListLine line;

  const _LineRow({required this.line});

  @override
  Widget build(BuildContext context) {
    final priceText = line.quantity > 1
        ? '${line.quantity}x ${formatBrl(line.price)} = ${formatBrl(line.lineTotal)}'
        : formatBrl(line.price);

    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 3.0),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(line.productName, style: const TextStyle(fontSize: 12, color: _ink)),
                if (line.isSubstitute)
                  Text(
                    line.brand.isEmpty ? 'outra marca' : 'outra marca: ${line.brand}',
                    style: const TextStyle(fontSize: 10, color: _amber, fontWeight: FontWeight.w600),
                  ),
              ],
            ),
          ),
          const SizedBox(width: 8),
          Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Text(
                priceText,
                style: TextStyle(fontSize: 12, fontWeight: FontWeight.bold, color: line.isCheapest ? _blue : _ink),
              ),
              if (line.isCheapest) ...[
                const SizedBox(width: 4),
                const Icon(Icons.check_circle_outline, color: _blue, size: 12),
              ],
            ],
          ),
        ],
      ),
    );
  }
}
