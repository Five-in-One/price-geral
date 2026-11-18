import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:url_launcher/url_launcher.dart';

import '../models/models.dart';
import '../theme/app_theme.dart';
import 'widgets.dart';

const _blue = AppColors.brand;
const _ink = AppColors.ink;
const _grey = AppColors.grayLight;

/// Google Maps search for [market]'s address (it opens the Maps app when
/// installed, the browser otherwise).
Uri mapsUriFor(Market market) {
  final query = market.fullAddress.isEmpty ? market.name : '${market.name}, ${market.fullAddress}';
  return Uri.https('www.google.com', '/maps/search/', {'api': '1', 'query': query});
}

/// One line with a pin: "Rua Félix Pacheco, 965 · Centro Sul". Nothing when
/// the market has no address.
class MarketAddressLine extends StatelessWidget {
  final Market market;
  final Color color;
  final int maxLines;

  const MarketAddressLine({super.key, required this.market, this.color = _grey, this.maxLines = 1});

  @override
  Widget build(BuildContext context) {
    if (!market.hasAddress) return const SizedBox.shrink();
    final text = [market.streetLine, market.neighborhood].where((s) => s.isNotEmpty).join(' · ');
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.only(top: 1),
          child: Icon(Icons.place_outlined, size: 13, color: color),
        ),
        const SizedBox(width: 3),
        Flexible(
          child: Text(
            text.isEmpty ? market.regionLine : text,
            maxLines: maxLines,
            overflow: TextOverflow.ellipsis,
            style: TextStyle(fontSize: 12, color: color),
          ),
        ),
      ],
    );
  }
}

/// Opens a sheet with [market]'s full address and ways to get there.
/// [distanceKm] overrides the market's own distance (the comparison has its own).
Future<void> showMarketDetails(BuildContext context, Market market, {double? distanceKm}) {
  return showModalBottomSheet<void>(
    context: context,
    showDragHandle: true,
    // Sized to its content (not the default 9/16 of the screen), scrolling
    // when that doesn't fit, e.g. in landscape.
    isScrollControlled: true,
    builder: (_) => _MarketDetailsSheet(market: market, distanceKm: distanceKm ?? market.distance),
  );
}

class _MarketDetailsSheet extends StatelessWidget {
  final Market market;
  final double? distanceKm;

  const _MarketDetailsSheet({required this.market, this.distanceKm});

  Future<void> _openMap(BuildContext context) async {
    final messenger = ScaffoldMessenger.of(context);
    final opened = await launchUrl(mapsUriFor(market), mode: LaunchMode.externalApplication);
    if (!opened) {
      messenger.showSnackBar(const SnackBar(content: Text('Não foi possível abrir o mapa.')));
    }
  }

  Future<void> _copy(BuildContext context) async {
    final messenger = ScaffoldMessenger.of(context);
    await Clipboard.setData(ClipboardData(text: market.fullAddress));
    messenger.showSnackBar(
      const SnackBar(content: Text('Endereço copiado.'), duration: Duration(seconds: 2)),
    );
  }

  @override
  Widget build(BuildContext context) {
    return SafeArea(
      child: SingleChildScrollView(
        padding: const EdgeInsets.fromLTRB(24, 0, 24, 20),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Row(
              children: [
                MarketAvatar(
                  initials: market.initials,
                  primaryColor: market.primaryColor,
                  backgroundColor: market.backgroundColor,
                  size: 52,
                ),
                const SizedBox(width: 14),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        market.name,
                        style: const TextStyle(fontSize: 17, fontWeight: FontWeight.w900, color: _ink),
                      ),
                      if (distanceKm != null) ...[
                        const SizedBox(height: 2),
                        Text('${formatKm(distanceKm!)} de você', style: const TextStyle(fontSize: 13, color: _grey)),
                      ],
                    ],
                  ),
                ),
              ],
            ),
            const SizedBox(height: 22),
            const Text(
              'ENDEREÇO',
              style: TextStyle(fontSize: 11, fontWeight: FontWeight.bold, color: _grey, letterSpacing: 0.5),
            ),
            const SizedBox(height: 6),
            if (market.hasAddress) ...[
              if (market.streetLine.isNotEmpty)
                Text(market.streetLine, style: const TextStyle(fontSize: 15, fontWeight: FontWeight.w600, color: _ink)),
              if (market.regionLine.isNotEmpty)
                Text(market.regionLine, style: const TextStyle(fontSize: 14, color: _ink)),
              if (market.cep.isNotEmpty)
                Text('CEP ${market.cep}', style: const TextStyle(fontSize: 13, color: _grey)),
            ] else
              const Text(
                'Este mercado ainda não cadastrou o endereço.',
                style: TextStyle(fontSize: 14, color: _grey),
              ),
            const SizedBox(height: 20),
            ElevatedButton.icon(
              onPressed: () => _openMap(context),
              icon: const Icon(Icons.map_outlined, size: 18),
              label: Text(market.hasAddress ? 'Abrir no mapa' : 'Procurar no mapa'),
            ),
            if (market.hasAddress) ...[
              const SizedBox(height: 8),
              OutlinedButton.icon(
                onPressed: () => _copy(context),
                icon: const Icon(Icons.copy_outlined, size: 18),
                label: const Text('Copiar endereço'),
              ),
            ],
          ],
        ),
      ),
    );
  }
}

/// A small "Ver endereço" link that opens [showMarketDetails].
class MarketAddressButton extends StatelessWidget {
  final Market market;
  final double? distanceKm;
  final Color color;

  const MarketAddressButton({super.key, required this.market, this.distanceKm, this.color = _blue});

  @override
  Widget build(BuildContext context) {
    return InkWell(
      onTap: () => showMarketDetails(context, market, distanceKm: distanceKm),
      borderRadius: BorderRadius.circular(6),
      child: Padding(
        padding: const EdgeInsets.symmetric(vertical: 4, horizontal: 2),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.place_outlined, size: 14, color: color),
            const SizedBox(width: 3),
            Text(
              'Ver endereço',
              style: TextStyle(fontSize: 12, fontWeight: FontWeight.bold, color: color),
            ),
          ],
        ),
      ),
    );
  }
}
