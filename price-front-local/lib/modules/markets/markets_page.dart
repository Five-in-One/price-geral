import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../core/models/models.dart';
import '../../core/services/app_state.dart';
import '../../core/components/market_address.dart';
import '../../core/components/radius_selector.dart';
import '../../core/components/ui.dart';
import '../../core/components/widgets.dart';
import '../../core/theme/app_theme.dart';
import '../dashboard/dashboard_page.dart';

class MarketsPage extends StatefulWidget {
  const MarketsPage({super.key});

  @override
  State<MarketsPage> createState() => _MarketsPageState();
}

class _MarketsPageState extends State<MarketsPage> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<AppState>().loadMarkets();
    });
  }

  /// "Encontramos N mercados a até X km de você." — X being the farthest
  /// market whose distance is known.
  String _summary(List<Market> markets) {
    if (markets.isEmpty) return 'Nenhum mercado nessa distância. Aumente a distância para ver mais.';
    final distances = markets.map((m) => m.distance).whereType<double>();
    final label = markets.length == 1 ? 'mercado' : 'mercados';
    if (distances.isEmpty) return 'Encontramos ${markets.length} $label.';
    final farthest = distances.reduce((a, b) => a > b ? a : b);
    final farthestLabel = formatKm(farthest < 0.1 ? 0.1 : farthest);
    return 'Encontramos ${markets.length} $label a até $farthestLabel de você.';
  }

  @override
  Widget build(BuildContext context) {
    final appState = context.watch<AppState>();
    final markets = appState.markets;
    final isLoading = appState.isLoadingMarkets;
    final error = appState.marketsError;

    return Scaffold(
      body: SafeArea(
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Expanded(
              child: ListView(
                padding: const EdgeInsets.fromLTRB(20, 12, 20, 12),
                children: [
                  // Header logo
                  const PriceLogoHeader(),
                  const SizedBox(height: 18),

                  HeroBanner(
                    eyebrow: 'Mercados',
                    emoji: '📍',
                    title: appState.userLocation != null ? 'Mercados' : 'Você está em',
                    highlight: appState.userLocation != null ? 'perto de você' : 'Teresina',
                    subtitle: Text(isLoading ? 'Buscando mercados perto de você...' : _summary(markets)),
                  ),
                  const SizedBox(height: 16),

                  // How far to look for markets
                  const AppCard(child: RadiusSelector()),
                  const SizedBox(height: 22),

                  if (isLoading)
                    const Padding(
                      padding: EdgeInsets.symmetric(vertical: 40),
                      child: Center(child: CircularProgressIndicator()),
                    )
                  else if (error != null)
                    ErrorRetry(
                      message: error,
                      onRetry: () => context.read<AppState>().loadMarkets(),
                    )
                  else ...[
                    if (markets.isNotEmpty)
                      Padding(
                        padding: const EdgeInsets.only(left: 4, bottom: 10),
                        child: Text(
                          'MERCADOS (${markets.length})',
                          style: const TextStyle(fontSize: 11, fontWeight: FontWeight.w800, color: AppColors.grayLight, letterSpacing: 0.8),
                        ),
                      ),
                    for (final market in markets)
                      Padding(
                        padding: const EdgeInsets.only(bottom: 12),
                        // Tapping a market shows its address and a way to get there.
                        child: Material(
                          color: Colors.white,
                          clipBehavior: Clip.antiAlias,
                          shape: RoundedRectangleBorder(
                            borderRadius: BorderRadius.circular(20),
                            side: const BorderSide(color: AppColors.border),
                          ),
                          child: InkWell(
                            onTap: () => showMarketDetails(context, market),
                            child: Padding(
                              padding: const EdgeInsets.all(14),
                              child: Row(
                                children: [
                                  MarketAvatar(
                                    initials: market.initials,
                                    primaryColor: market.primaryColor,
                                    backgroundColor: market.backgroundColor,
                                    size: 46,
                                  ),
                                  const SizedBox(width: 14),
                                  Expanded(
                                    child: Column(
                                      crossAxisAlignment: CrossAxisAlignment.start,
                                      children: [
                                        Text(
                                          market.name,
                                          style: const TextStyle(
                                            fontSize: 15,
                                            fontWeight: FontWeight.w800,
                                            color: AppColors.ink,
                                          ),
                                        ),
                                        if (market.hasAddress) ...[
                                          const SizedBox(height: 4),
                                          MarketAddressLine(market: market, maxLines: 2),
                                        ],
                                      ],
                                    ),
                                  ),
                                  const SizedBox(width: 8),
                                  if (market.distance != null)
                                    AppChip(formatKm(market.distance!), icon: Icons.near_me_rounded)
                                  else
                                    const Icon(Icons.chevron_right_rounded, color: AppColors.grayLight),
                                ],
                              ),
                            ),
                          ),
                        ),
                      ),
                  ],
                ],
              ),
            ),

            // Ver produtos e preços Button
            Container(
              padding: const EdgeInsets.fromLTRB(20, 12, 20, 16),
              decoration: BoxDecoration(
                color: Colors.white,
                borderRadius: const BorderRadius.vertical(top: Radius.circular(24)),
                boxShadow: [BoxShadow(color: const Color(0xFF1E3C78).withValues(alpha: 0.08), blurRadius: 24, offset: const Offset(0, -6))],
              ),
              child: ElevatedButton(
                onPressed: () {
                  Navigator.of(context).pushAndRemoveUntil(
                    MaterialPageRoute(builder: (_) => const DashboardPage()),
                    (route) => false,
                  );
                },
                child: const Row(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    Text('Ver produtos e preços'),
                    SizedBox(width: 8),
                    Icon(Icons.arrow_forward, size: 18),
                  ],
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
