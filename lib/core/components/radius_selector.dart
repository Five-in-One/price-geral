import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../services/app_state.dart';
import '../theme/app_theme.dart';
import 'ui.dart';

/// Lets the person choose how far the app looks for markets ("até 5 km").
/// Dragging only moves the label; the markets and the comparison reload
/// once, when the finger is lifted.
class RadiusSelector extends StatefulWidget {
  const RadiusSelector({super.key});

  @override
  State<RadiusSelector> createState() => _RadiusSelectorState();
}

class _RadiusSelectorState extends State<RadiusSelector> {
  /// The value under the finger while dragging; null otherwise.
  double? _dragging;

  @override
  Widget build(BuildContext context) {
    final appState = context.watch<AppState>();
    final value = _dragging ?? appState.radiusKm;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Row(
          children: [
            const IconTile(Icons.radar, size: 38),
            const SizedBox(width: 12),
            const Expanded(
              child: Text(
                'Distância dos mercados',
                style: TextStyle(fontSize: 15, fontWeight: FontWeight.w800, color: AppColors.ink),
              ),
            ),
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
              decoration: BoxDecoration(color: AppColors.accentBg, borderRadius: BorderRadius.circular(999)),
              child: Text(
                'até ${value.round()} km',
                style: const TextStyle(fontSize: 13, fontWeight: FontWeight.w800, color: AppColors.brand),
              ),
            ),
          ],
        ),
        Slider(
          value: value,
          min: AppState.minRadiusKm,
          max: AppState.maxRadiusKm,
          divisions: (AppState.maxRadiusKm - AppState.minRadiusKm).round(),
          label: '${value.round()} km',
          semanticFormatterCallback: (v) => 'até ${v.round()} quilômetros',
          onChanged: (v) => setState(() => _dragging = v),
          onChangeEnd: (v) {
            setState(() => _dragging = null);
            appState.setRadiusKm(v);
          },
        ),
        const Text(
          'Só aparecem mercados até essa distância de você. A comparação da lista também usa esse limite.',
          style: TextStyle(fontSize: 12, height: 1.4, color: AppColors.gray),
        ),
      ],
    );
  }
}
