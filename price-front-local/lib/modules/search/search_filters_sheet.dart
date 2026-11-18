import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../../core/models/models.dart';
import '../../core/models/product_search.dart';
import '../../core/theme/app_theme.dart';

/// Bottom sheet with the search filters: nearby markets only, which markets,
/// and a price range. Pops the chosen [SearchFilters] on "Aplicar" (or empty
/// ones on "Limpar"); dismissing it changes nothing.
class SearchFiltersSheet extends StatefulWidget {
  final SearchFilters initial;
  final List<Market> markets;
  final bool hasLocation;
  final double radiusKm;

  const SearchFiltersSheet({
    super.key,
    required this.initial,
    required this.markets,
    required this.hasLocation,
    required this.radiusKm,
  });

  @override
  State<SearchFiltersSheet> createState() => _SearchFiltersSheetState();
}

class _SearchFiltersSheetState extends State<SearchFiltersSheet> {
  late final Set<String> _marketIds = {...widget.initial.marketIds};
  late bool _nearbyOnly = widget.initial.nearbyOnly && widget.hasLocation;
  late final _min = TextEditingController(text: _priceText(widget.initial.minPrice));
  late final _max = TextEditingController(text: _priceText(widget.initial.maxPrice));
  String? _priceError;

  static String _priceText(double? v) => v == null ? '' : v.toStringAsFixed(2).replaceAll('.', ',');

  /// "12,50" or "12.5" -> 12.5; "" -> null.
  static double? _parsePrice(String text) {
    final t = text.trim().replaceAll(',', '.');
    return t.isEmpty ? null : double.tryParse(t);
  }

  @override
  void dispose() {
    _min.dispose();
    _max.dispose();
    super.dispose();
  }

  void _apply() {
    final min = _parsePrice(_min.text);
    final max = _parsePrice(_max.text);
    if ((_min.text.trim().isNotEmpty && min == null) || (_max.text.trim().isNotEmpty && max == null)) {
      setState(() => _priceError = 'Digite um valor como 12,50.');
      return;
    }
    if (min != null && max != null && min > max) {
      setState(() => _priceError = 'O mínimo não pode ser maior que o máximo.');
      return;
    }
    Navigator.pop(context, SearchFilters(marketIds: _marketIds, minPrice: min, maxPrice: max, nearbyOnly: _nearbyOnly));
  }

  @override
  Widget build(BuildContext context) {
    final priceInput = <TextInputFormatter>[FilteringTextInputFormatter.allow(RegExp(r'[0-9.,]'))];

    return Padding(
      // Keeps the price fields above the keyboard.
      padding: EdgeInsets.only(bottom: MediaQuery.viewInsetsOf(context).bottom),
      child: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.fromLTRB(20, 0, 20, 16),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const Text(
                'Filtros',
                style: TextStyle(fontSize: 16, fontWeight: FontWeight.w800, color: AppColors.ink),
              ),
              const SizedBox(height: 8),
              SwitchListTile(
                key: const ValueKey('filter-nearby'),
                contentPadding: EdgeInsets.zero,
                value: _nearbyOnly,
                onChanged: widget.hasLocation ? (v) => setState(() => _nearbyOnly = v) : null,
                title: Text('Só mercados até ${widget.radiusKm.round()} km'),
                subtitle: Text(
                  widget.hasLocation
                      ? 'Usa a sua localização e o raio escolhido no Perfil.'
                      : 'Atualize sua localização no topo da busca para usar.',
                ),
              ),
              if (widget.markets.isNotEmpty) ...[
                const SizedBox(height: 8),
                const Text(
                  'Mercados',
                  style: TextStyle(fontSize: 13, fontWeight: FontWeight.w800, color: AppColors.ink),
                ),
                const SizedBox(height: 8),
                Wrap(
                  spacing: 8,
                  runSpacing: 8,
                  children: [
                    for (final m in widget.markets)
                      FilterChip(
                        label: Text(m.name),
                        selected: _marketIds.contains(m.id),
                        onSelected: (on) => setState(() => on ? _marketIds.add(m.id) : _marketIds.remove(m.id)),
                      ),
                  ],
                ),
              ],
              const SizedBox(height: 16),
              const Text(
                'Preço (R\$)',
                style: TextStyle(fontSize: 13, fontWeight: FontWeight.w800, color: AppColors.ink),
              ),
              const SizedBox(height: 8),
              Row(
                children: [
                  Expanded(
                    child: TextField(
                      key: const ValueKey('filter-min-price'),
                      controller: _min,
                      keyboardType: const TextInputType.numberWithOptions(decimal: true),
                      inputFormatters: priceInput,
                      decoration: const InputDecoration(labelText: 'Mínimo', prefixText: 'R\$ '),
                    ),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: TextField(
                      key: const ValueKey('filter-max-price'),
                      controller: _max,
                      keyboardType: const TextInputType.numberWithOptions(decimal: true),
                      inputFormatters: priceInput,
                      decoration: const InputDecoration(labelText: 'Máximo', prefixText: 'R\$ '),
                    ),
                  ),
                ],
              ),
              if (_priceError != null) ...[
                const SizedBox(height: 6),
                Text(_priceError!, style: TextStyle(fontSize: 12, color: Theme.of(context).colorScheme.error)),
              ],
              const SizedBox(height: 20),
              Row(
                children: [
                  Expanded(
                    child: OutlinedButton(
                      onPressed: () => Navigator.pop(context, const SearchFilters()),
                      child: const Text('Limpar'),
                    ),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: ElevatedButton(
                      key: const ValueKey('filter-apply'),
                      onPressed: _apply,
                      child: const Text('Aplicar'),
                    ),
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }
}
