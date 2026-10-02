import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../core/services/app_state.dart';
import '../../core/theme/app_theme.dart';
import '../search/search_page.dart';
import '../cart/cart_page.dart';
import '../comparison/comparison_page.dart';
// TODO: reativar junto com a aba Economia (comentada por enquanto).
// import '../economy/economy_page.dart';
import '../user/user_page.dart';

class DashboardPage extends StatelessWidget {
  const DashboardPage({super.key});

  @override
  Widget build(BuildContext context) {
    final appState = Provider.of<AppState>(context);
    final activeIndex = appState.currentTabIndex;

    // Pages mapping the tabs
    final List<Widget> pages = [
      const SearchPage(),
      const CartPage(),
      const ComparisonPage(),
      // const EconomyPage(),
      const UserPage(),
    ];

    final int totalCartItems = appState.totalItemsCount;

    Widget listIcon(IconData icon) => Badge(
          isLabelVisible: totalCartItems > 0,
          backgroundColor: AppColors.brand,
          label: Text('$totalCartItems', style: const TextStyle(fontSize: 9, fontWeight: FontWeight.w800)),
          child: Icon(icon),
        );

    return Scaffold(
      body: IndexedStack(
        index: activeIndex,
        children: pages,
      ),
      bottomNavigationBar: Container(
        decoration: BoxDecoration(
          color: Colors.white,
          borderRadius: const BorderRadius.vertical(top: Radius.circular(24)),
          boxShadow: [
            BoxShadow(color: const Color(0xFF1E3C78).withValues(alpha: 0.08), blurRadius: 24, offset: const Offset(0, -6)),
          ],
        ),
        child: ClipRRect(
          borderRadius: const BorderRadius.vertical(top: Radius.circular(24)),
          child: NavigationBar(
            selectedIndex: activeIndex,
            onDestinationSelected: (index) => appState.setTabIndex(index),
            labelBehavior: NavigationDestinationLabelBehavior.alwaysShow,
            destinations: [
              const NavigationDestination(
                icon: Icon(Icons.search_outlined),
                selectedIcon: Icon(Icons.search),
                label: 'Busca',
              ),
              NavigationDestination(
                icon: listIcon(Icons.shopping_bag_outlined),
                selectedIcon: listIcon(Icons.shopping_bag),
                label: 'Lista',
              ),
              const NavigationDestination(
                icon: Icon(Icons.balance_outlined),
                selectedIcon: Icon(Icons.balance),
                label: 'Comparar',
              ),
              // Aba Economia (comentada por enquanto; reative junto com a página acima)
              // const NavigationDestination(
              //   icon: Icon(Icons.trending_up_outlined),
              //   selectedIcon: Icon(Icons.trending_up),
              //   label: 'Economia',
              // ),
              const NavigationDestination(
                icon: Icon(Icons.person_outline),
                selectedIcon: Icon(Icons.person),
                label: 'Perfil',
              ),
            ],
          ),
        ),
      ),
    );
  }
}
