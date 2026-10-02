import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../core/services/app_state.dart';
import '../../core/components/ui.dart';
import '../../core/components/widgets.dart';
import '../../core/theme/app_theme.dart';

class CartPage extends StatelessWidget {
  const CartPage({super.key});

  void _confirmClear(BuildContext context, AppState appState) {
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Limpar Lista'),
        content: const Text('Deseja remover todos os itens da sua lista?'),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(),
            child: const Text('Cancelar', style: TextStyle(color: AppColors.gray)),
          ),
          TextButton(
            onPressed: () {
              appState.clearCart();
              Navigator.of(ctx).pop();
            },
            child: const Text('Limpar', style: TextStyle(color: AppColors.red)),
          ),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final appState = Provider.of<AppState>(context);
    final cartItems = appState.cartItems;
    final count = appState.totalItemsCount;

    return Scaffold(
      body: SafeArea(
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Expanded(
              child: ListView(
                padding: const EdgeInsets.fromLTRB(20, 12, 20, 16),
                children: [
                  HeroBanner(
                    eyebrow: 'Minha lista',
                    title: 'Lista de',
                    highlight: 'compras',
                    emoji: '🧺',
                    children: [
                      Row(
                        children: [
                          StatChip(count == 1 ? '1 item' : '$count itens'),
                          const Spacer(),
                          // Clear the whole list (asks first)
                          Material(
                            color: Colors.white,
                            shape: const StadiumBorder(side: BorderSide(color: AppColors.border)),
                            clipBehavior: Clip.antiAlias,
                            child: InkWell(
                              onTap: cartItems.isEmpty ? null : () => _confirmClear(context, appState),
                              child: Padding(
                                padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 7),
                                child: Row(
                                  mainAxisSize: MainAxisSize.min,
                                  children: [
                                    Icon(
                                      Icons.delete_outline,
                                      size: 15,
                                      color: cartItems.isEmpty ? AppColors.border : AppColors.red,
                                    ),
                                    const SizedBox(width: 4),
                                    Text(
                                      'Limpar',
                                      style: TextStyle(
                                        fontSize: 12,
                                        fontWeight: FontWeight.w700,
                                        color: cartItems.isEmpty ? AppColors.grayLight : AppColors.red,
                                      ),
                                    ),
                                  ],
                                ),
                              ),
                            ),
                          ),
                        ],
                      ),
                    ],
                  ),
                  const SizedBox(height: 18),

                  if (cartItems.isEmpty)
                    EmptyState(
                      icon: Icons.shopping_basket_outlined,
                      title: 'Sua lista está vazia.',
                      message: 'Adicione produtos na Busca e compare o total nos mercados perto de você.',
                      action: ElevatedButton.icon(
                        onPressed: () => appState.setTabIndex(0),
                        icon: const Icon(Icons.search, size: 18),
                        label: const Text('Buscar produtos'),
                      ),
                    )
                  else
                    for (final item in cartItems)
                      Padding(
                        padding: const EdgeInsets.only(bottom: 12),
                        child: Container(
                          padding: const EdgeInsets.fromLTRB(14, 12, 14, 12),
                          decoration: BoxDecoration(
                            color: Colors.white,
                            borderRadius: BorderRadius.circular(20),
                            border: Border.all(color: AppColors.border),
                          ),
                          child: Row(
                            children: [
                              Container(
                                width: 42,
                                height: 42,
                                alignment: Alignment.center,
                                decoration: BoxDecoration(color: AppColors.accentBg, borderRadius: BorderRadius.circular(13)),
                                child: Text(
                                  item.product.name.isEmpty ? '?' : item.product.name[0].toUpperCase(),
                                  style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w900, color: AppColors.brand),
                                ),
                              ),
                              const SizedBox(width: 12),
                              // Name & details
                              Expanded(
                                child: Column(
                                  crossAxisAlignment: CrossAxisAlignment.start,
                                  children: [
                                    Text(
                                      item.product.name,
                                      style: const TextStyle(
                                        fontSize: 14,
                                        fontWeight: FontWeight.w800,
                                        color: AppColors.ink,
                                      ),
                                    ),
                                    const SizedBox(height: 3),
                                    Text(
                                      'R\$ ${item.product.defaultPrice.toStringAsFixed(2).replaceAll('.', ',')}/${item.product.unit}',
                                      style: const TextStyle(fontSize: 11, color: AppColors.grayLight),
                                    ),
                                    const SizedBox(height: 8),
                                    QuantitySelector(
                                      quantity: item.quantity,
                                      onIncrement: () => appState.updateQuantity(item.product, 1),
                                      onDecrement: () => appState.updateQuantity(item.product, -1),
                                      showIfZero: true,
                                    ),
                                  ],
                                ),
                              ),
                              const SizedBox(width: 10),

                              // Total Price
                              Text(
                                'R\$ ${item.totalPrice.toStringAsFixed(2).replaceAll('.', ',')}',
                                textAlign: TextAlign.end,
                                style: const TextStyle(
                                  fontSize: 15,
                                  fontWeight: FontWeight.w900,
                                  color: AppColors.ink,
                                ),
                              ),
                            ],
                          ),
                        ),
                      ),
                ],
              ),
            ),

            // Sticky footer
            if (cartItems.isNotEmpty)
              Container(
                padding: const EdgeInsets.fromLTRB(20.0, 16.0, 20.0, 16.0),
                decoration: BoxDecoration(
                  color: Colors.white,
                  borderRadius: const BorderRadius.vertical(top: Radius.circular(24)),
                  boxShadow: [
                    BoxShadow(color: const Color(0xFF1E3C78).withValues(alpha: 0.08), blurRadius: 24, offset: const Offset(0, -6)),
                  ],
                ),
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    // Total estimate row
                    Row(
                      mainAxisAlignment: MainAxisAlignment.spaceBetween,
                      children: [
                        Text(
                          'Total Estimado ($count itens)',
                          style: const TextStyle(
                            fontSize: 13,
                            color: AppColors.gray,
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                        Text(
                          'R\$ ${appState.defaultTotalPrice.toStringAsFixed(2).replaceAll('.', ',')}',
                          style: const TextStyle(
                            fontSize: 22,
                            fontWeight: FontWeight.w900,
                            color: AppColors.ink,
                            letterSpacing: -0.5,
                          ),
                        ),
                      ],
                    ),
                    const SizedBox(height: 14),

                    // "Comparar em 5 mercados" button
                    ElevatedButton(
                      onPressed: () {
                        // Switch to the Compare Tab (index 2)
                        appState.setTabIndex(2);
                      },
                      child: Row(
                        mainAxisAlignment: MainAxisAlignment.center,
                        children: [
                          const Icon(Icons.compare_arrows, size: 18),
                          const SizedBox(width: 8),
                          Text('Comparar em ${appState.markets.isNotEmpty ? appState.markets.length : 5} mercados'),
                        ],
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
