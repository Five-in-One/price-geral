import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'core/services/app_state.dart';
import 'core/services/location_service.dart';
import 'core/components/location_feedback.dart';
import 'core/components/ui.dart';
import 'core/components/widgets.dart';
import 'core/theme/app_theme.dart';
import 'modules/markets/markets_page.dart';

class SplashPage extends StatefulWidget {
  const SplashPage({super.key});

  @override
  State<SplashPage> createState() => _SplashPageState();
}

class _SplashPageState extends State<SplashPage> {
  bool _isLoading = false;

  Future<void> _handleLocationClick(BuildContext context) async {
    setState(() {
      _isLoading = true;
    });

    final appState = Provider.of<AppState>(context, listen: false);
    final navigator = Navigator.of(context);
    final messenger = ScaffoldMessenger.of(context);

    try {
      // Real GPS position; the markets list then comes back with the
      // distance from here to each market.
      await appState.detectLocation();
      if (!mounted) return;
      appState.setAddress('Minha localização');
      navigator.push(
        MaterialPageRoute(builder: (_) => const MarketsPage()),
      );
    } on LocationException catch (e) {
      if (!mounted) return;
      showLocationError(messenger, appState, e);
    } finally {
      if (mounted) {
        setState(() {
          _isLoading = false;
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: SafeArea(
        child: Stack(
          children: [
            // Loose shapes in the corners, like the web panels.
            Positioned(
              right: -70,
              top: -60,
              child: Container(
                width: 170,
                height: 170,
                decoration: const BoxDecoration(color: AppColors.accentBg, shape: BoxShape.circle),
              ),
            ),
            Positioned(
              left: -60,
              bottom: 120,
              child: Container(
                width: 120,
                height: 120,
                decoration: BoxDecoration(shape: BoxShape.circle, border: Border.all(color: AppColors.heroRing, width: 2)),
              ),
            ),
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 24.0, vertical: 16.0),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  // Header logo
                  const PriceLogoHeader(),
                  const SizedBox(height: 28),

                  // Map Graphic Widget
                  const StylizedMapWidget(),
                  const SizedBox(height: 28),

                  // Welcome details
                  const Align(alignment: Alignment.centerLeft, child: EyebrowPill('Compare e economize')),
                  const SizedBox(height: 14),
                  const Text.rich(
                    TextSpan(
                      children: [
                        TextSpan(text: 'Os melhores preços\n'),
                        TextSpan(text: 'perto de você', style: TextStyle(color: AppColors.brand)),
                      ],
                    ),
                    style: TextStyle(
                      fontSize: 30,
                      fontWeight: FontWeight.w900,
                      height: 1.1,
                      color: AppColors.ink,
                      letterSpacing: -0.8,
                    ),
                  ),
                  const SizedBox(height: 12),
                  const Text(
                    'Compare a sua lista de compras nos mercados do seu bairro e veja onde gastar menos.',
                    style: TextStyle(
                      fontSize: 14,
                      height: 1.45,
                      color: AppColors.gray,
                    ),
                  ),
                  const Spacer(),

                  // Primary Button
                  ElevatedButton(
                    onPressed: _isLoading ? null : () => _handleLocationClick(context),
                    style: ElevatedButton.styleFrom(
                      padding: const EdgeInsets.symmetric(vertical: 18),
                      // Stays blue while the GPS is read, so the white spinner shows.
                      disabledBackgroundColor: AppColors.brand.withValues(alpha: 0.75),
                    ),
                    child: _isLoading
                        ? const SizedBox(
                            height: 20,
                            width: 20,
                            child: CircularProgressIndicator(
                              color: Colors.white,
                              strokeWidth: 2,
                            ),
                          )
                        : const Row(
                            mainAxisAlignment: MainAxisAlignment.center,
                            children: [
                              Icon(Icons.gps_fixed, size: 18),
                              SizedBox(width: 8),
                              Text(
                                'Usar minha localização',
                                style: TextStyle(
                                  fontSize: 16,
                                  fontWeight: FontWeight.bold,
                                ),
                              ),
                            ],
                          ),
                  ),
                  const SizedBox(height: 10),

                  // Secondary Manual action
                  TextButton(
                    onPressed: () {
                      // Instant bypass for testing convenience
                      Provider.of<AppState>(context, listen: false).setAddress('Teresina');
                      Navigator.of(context).push(
                        MaterialPageRoute(builder: (_) => const MarketsPage()),
                      );
                    },
                    child: const Text(
                      'Ver todos os mercados do meu município',
                      style: TextStyle(
                        color: AppColors.gray,
                        fontSize: 14,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}
