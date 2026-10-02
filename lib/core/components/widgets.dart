import 'package:flutter/material.dart';
import 'package:flutter_svg/flutter_svg.dart';

import '../theme/app_theme.dart';
import 'ui.dart';

/// The "Price" wordmark with its logo in a soft blue tile, used as the small
/// header on the splash and markets screens.
class PriceLogoHeader extends StatelessWidget {
  final double iconSize;
  final double fontSize;

  const PriceLogoHeader({super.key, this.iconSize = 22, this.fontSize = 18});

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisAlignment: MainAxisAlignment.center,
      children: [
        Container(
          width: iconSize + 14,
          height: iconSize + 14,
          alignment: Alignment.center,
          decoration: BoxDecoration(color: AppColors.accentBg, borderRadius: BorderRadius.circular(11)),
          child: SvgPicture.asset('assets/logo-price.svg', width: iconSize, height: iconSize),
        ),
        const SizedBox(width: 10),
        Text(
          'Price',
          style: TextStyle(
            fontSize: fontSize,
            fontWeight: FontWeight.w900,
            color: AppColors.brand,
            letterSpacing: -0.5,
          ),
        ),
      ],
    );
  }
}

/// The splash illustration: a neighbourhood map in a rounded card, with the
/// person's pin in the middle and floating market/price chips around it.
class StylizedMapWidget extends StatelessWidget {
  const StylizedMapWidget({super.key});

  @override
  Widget build(BuildContext context) {
    Widget chip(String emoji) => Container(
          width: 40,
          height: 40,
          alignment: Alignment.center,
          decoration: BoxDecoration(
            color: Colors.white,
            shape: BoxShape.circle,
            boxShadow: [BoxShadow(color: Colors.black.withValues(alpha: 0.12), blurRadius: 14, offset: const Offset(0, 6))],
          ),
          child: Text(emoji, style: const TextStyle(fontSize: 18)),
        );

    return Container(
      height: 220,
      width: double.infinity,
      decoration: BoxDecoration(
        color: AppColors.heroBg,
        borderRadius: BorderRadius.circular(28),
        border: Border.all(color: AppColors.border),
      ),
      clipBehavior: Clip.antiAlias,
      child: Stack(
        children: [
          Positioned.fill(child: CustomPaint(painter: MapPainter())),
          Center(
            child: Container(
              width: 150,
              height: 150,
              decoration: BoxDecoration(shape: BoxShape.circle, border: Border.all(color: AppColors.heroRing, width: 2)),
            ),
          ),
          Center(
            child: Container(
              width: 96,
              height: 96,
              decoration: BoxDecoration(color: AppColors.brand.withValues(alpha: 0.14), shape: BoxShape.circle),
              child: Center(
                child: Container(
                  width: 48,
                  height: 48,
                  decoration: BoxDecoration(
                    color: AppColors.brand,
                    shape: BoxShape.circle,
                    border: Border.all(color: Colors.white, width: 3),
                    boxShadow: [BoxShadow(color: AppColors.brand.withValues(alpha: 0.35), blurRadius: 16, offset: const Offset(0, 6))],
                  ),
                  child: const Icon(Icons.location_on, color: Colors.white, size: 24),
                ),
              ),
            ),
          ),
          Positioned(left: 28, top: 26, child: chip('🏪')),
          Positioned(right: 34, top: 40, child: chip('🛒')),
          Positioned(left: 56, bottom: 26, child: chip('🥦')),
          Positioned(
            right: 22,
            bottom: 24,
            child: Container(
              padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
              decoration: BoxDecoration(
                color: Colors.white,
                borderRadius: BorderRadius.circular(14),
                boxShadow: [BoxShadow(color: Colors.black.withValues(alpha: 0.12), blurRadius: 14, offset: const Offset(0, 6))],
              ),
              child: const Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                mainAxisSize: MainAxisSize.min,
                children: [
                  Text('mais barato', style: TextStyle(fontSize: 9, fontWeight: FontWeight.w600, color: AppColors.grayLight)),
                  Text('pertinho', style: TextStyle(fontSize: 12, fontWeight: FontWeight.w900, color: AppColors.brand)),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class MapPainter extends CustomPainter {
  @override
  void paint(Canvas canvas, Size size) {
    final paint = Paint()
      ..color = const Color(0xFFD5E1F3)
      ..strokeWidth = 2.0;

    // Draw main grid streets
    final double midX = size.width / 2;
    final double midY = size.height / 2;

    // Vertical street (main axis)
    canvas.drawLine(Offset(midX, 0), Offset(midX, size.height), paint);
    // Horizontal street (main axis)
    canvas.drawLine(Offset(0, midY), Offset(size.width, midY), paint);

    // Parallel minor streets
    paint.strokeWidth = 1.0;
    paint.color = const Color(0xFFE2EAF6);

    // Vertical grid lines
    for (double i = midX - 80; i > 0; i -= 60) {
      canvas.drawLine(Offset(i, 0), Offset(i, size.height), paint);
    }
    for (double i = midX + 80; i < size.width; i += 60) {
      canvas.drawLine(Offset(i, 0), Offset(i, size.height), paint);
    }

    // Horizontal grid lines
    for (double i = midY - 60; i > 0; i -= 50) {
      canvas.drawLine(Offset(0, i), Offset(size.width, i), paint);
    }
    for (double i = midY + 60; i < size.height; i += 50) {
      canvas.drawLine(Offset(0, i), Offset(size.width, i), paint);
    }
  }

  @override
  bool shouldRepaint(covariant CustomPainter oldDelegate) => false;
}

class MarketAvatar extends StatelessWidget {
  final String initials;
  final Color primaryColor;
  final Color backgroundColor;
  final double size;

  const MarketAvatar({
    super.key,
    required this.initials,
    required this.primaryColor,
    required this.backgroundColor,
    this.size = 38.0,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      width: size,
      height: size,
      decoration: BoxDecoration(
        color: backgroundColor,
        borderRadius: BorderRadius.circular(size * 0.32),
      ),
      child: Center(
        child: Text(
          initials,
          style: TextStyle(
            color: primaryColor,
            fontSize: size * 0.36,
            fontWeight: FontWeight.w900,
          ),
        ),
      ),
    );
  }
}

class QuantitySelector extends StatelessWidget {
  final int quantity;
  final VoidCallback onDecrement;
  final VoidCallback onIncrement;
  final bool showIfZero;

  const QuantitySelector({
    super.key,
    required this.quantity,
    required this.onDecrement,
    required this.onIncrement,
    this.showIfZero = false,
  });

  @override
  Widget build(BuildContext context) {
    if (quantity == 0 && !showIfZero) {
      return GestureDetector(
        onTap: onIncrement,
        child: Container(
          width: 36,
          height: 36,
          decoration: BoxDecoration(
            color: AppColors.brand,
            shape: BoxShape.circle,
            boxShadow: [BoxShadow(color: AppColors.brand.withValues(alpha: 0.3), blurRadius: 10, offset: const Offset(0, 4))],
          ),
          child: const Icon(Icons.add, color: Colors.white, size: 20),
        ),
      );
    }

    return Container(
      padding: const EdgeInsets.all(3),
      decoration: BoxDecoration(
        color: AppColors.accentBg,
        borderRadius: BorderRadius.circular(999),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          GestureDetector(
            onTap: onDecrement,
            child: Container(
              width: 28,
              height: 28,
              decoration: const BoxDecoration(color: Colors.white, shape: BoxShape.circle),
              child: const Icon(Icons.remove, color: AppColors.brand, size: 16),
            ),
          ),
          SizedBox(
            width: 30,
            child: Text(
              '$quantity',
              textAlign: TextAlign.center,
              style: const TextStyle(fontSize: 14, fontWeight: FontWeight.w900, color: AppColors.ink),
            ),
          ),
          GestureDetector(
            onTap: onIncrement,
            child: Container(
              width: 28,
              height: 28,
              decoration: const BoxDecoration(color: AppColors.brand, shape: BoxShape.circle),
              child: const Icon(Icons.add, color: Colors.white, size: 16),
            ),
          ),
        ],
      ),
    );
  }
}

class CustomBarChart extends StatelessWidget {
  final List<ChartBarData> data;

  const CustomBarChart({super.key, required this.data});

  @override
  Widget build(BuildContext context) {
    const double maxHeight = 120.0;
    final double maxVal = data.isEmpty
        ? 1.0
        : data.map((d) => d.value).reduce((a, b) => a > b ? a : b);

    return Row(
      mainAxisAlignment: MainAxisAlignment.spaceBetween,
      crossAxisAlignment: CrossAxisAlignment.end,
      children: data.map((point) {
        final double height = (point.value / maxVal) * maxHeight;

        return Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(
              '${point.value.toInt()}',
              style: TextStyle(
                fontSize: 11,
                fontWeight: point.isActive ? FontWeight.bold : FontWeight.normal,
                color: point.isActive ? const Color(0xFF2A5CB9) : const Color(0xFF8E8E93),
              ),
            ),
            const SizedBox(height: 6),
            Container(
              width: 24,
              height: height,
              decoration: BoxDecoration(
                color: point.isActive
                    ? const Color(0xFF2A5CB9)
                    : const Color(0xFF2A5CB9).withValues(alpha: 0.3),
                borderRadius: const BorderRadius.only(
                  topLeft: Radius.circular(6),
                  topRight: Radius.circular(6),
                ),
              ),
            ),
            const SizedBox(height: 8),
            Text(
              point.label,
              style: TextStyle(
                fontSize: 11,
                color: point.isActive ? const Color(0xFF1C1C1E) : const Color(0xFF8E8E93),
                fontWeight: point.isActive ? FontWeight.bold : FontWeight.normal,
              ),
            ),
          ],
        );
      }).toList(),
    );
  }
}

/// A centered "something went wrong" state with a retry action, shown when
/// a network request to price-back fails.
class ErrorRetry extends StatelessWidget {
  final String message;
  final VoidCallback onRetry;

  const ErrorRetry({super.key, required this.message, required this.onRetry});

  @override
  Widget build(BuildContext context) {
    return EmptyState(
      icon: Icons.wifi_off_rounded,
      title: 'Algo deu errado',
      message: message,
      action: OutlinedButton.icon(
        onPressed: onRetry,
        icon: const Icon(Icons.refresh, size: 18),
        label: const Text('Tentar novamente'),
      ),
    );
  }
}

class ChartBarData {
  final String label;
  final double value;
  final bool isActive;

  ChartBarData({
    required this.label,
    required this.value,
    this.isActive = false,
  });
}
