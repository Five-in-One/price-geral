import 'package:flutter/material.dart';
import 'package:flutter_svg/flutter_svg.dart';

import '../theme/app_theme.dart';

/// Building blocks of the app's look (the same language as the web panels):
/// a hero banner with loose shapes, white rounded cards, section headers with
/// an icon tile, pill chips and friendly empty states.

/// A white rounded card with a soft shadow.
class AppCard extends StatelessWidget {
  final Widget child;
  final EdgeInsetsGeometry padding;
  final Color color;
  final Color borderColor;
  final double radius;

  const AppCard({
    super.key,
    required this.child,
    this.padding = const EdgeInsets.all(18),
    this.color = AppColors.surface,
    this.borderColor = AppColors.border,
    this.radius = 22,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: padding,
      decoration: BoxDecoration(
        color: color,
        borderRadius: BorderRadius.circular(radius),
        border: Border.all(color: borderColor),
        boxShadow: AppColors.softShadow,
      ),
      child: child,
    );
  }
}

/// "• PRICE" — the small uppercase label above a hero title.
class EyebrowPill extends StatelessWidget {
  final String text;

  const EyebrowPill(this.text, {super.key});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(999),
        boxShadow: AppColors.softShadow,
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(width: 6, height: 6, decoration: const BoxDecoration(color: AppColors.brand, shape: BoxShape.circle)),
          const SizedBox(width: 7),
          Flexible(
            child: Text(
              text.toUpperCase(),
              overflow: TextOverflow.ellipsis,
              style: const TextStyle(fontSize: 10, fontWeight: FontWeight.w800, letterSpacing: 1.6, color: AppColors.brand),
            ),
          ),
        ],
      ),
    );
  }
}

/// The banner at the top of each screen: eyebrow, a heavy two-tone title,
/// optional subtitle and extra content (chips, buttons), with the Price logo
/// on a blue disc and loose shapes in the top-right corner.
class HeroBanner extends StatelessWidget {
  final String eyebrow;
  final String title;

  /// Second part of the title, drawn in blue ("Lista de *compras*").
  final String highlight;
  final Widget? subtitle;
  final List<Widget> children;
  final String emoji;

  const HeroBanner({
    super.key,
    required this.eyebrow,
    required this.title,
    this.highlight = '',
    this.subtitle,
    this.children = const [],
    this.emoji = '🛒',
  });

  @override
  Widget build(BuildContext context) {
    return ClipRRect(
      borderRadius: BorderRadius.circular(28),
      child: Container(
        decoration: BoxDecoration(
          color: AppColors.heroBg,
          borderRadius: BorderRadius.circular(28),
          border: Border.all(color: AppColors.border),
        ),
        child: Stack(
          children: [
            Positioned(right: -34, top: -52, child: _circle(136, border: AppColors.heroRing)),
            Positioned(right: -20, top: -34, child: _circle(100, color: AppColors.brand)),
            Positioned(
              right: 20,
              top: 14,
              child: Transform.rotate(
                angle: -0.1,
                child: Container(
                  width: 46,
                  height: 46,
                  padding: const EdgeInsets.all(8),
                  decoration: BoxDecoration(
                    color: Colors.white,
                    borderRadius: BorderRadius.circular(15),
                    boxShadow: [BoxShadow(color: Colors.black.withValues(alpha: 0.18), blurRadius: 16, offset: const Offset(0, 6))],
                  ),
                  child: SvgPicture.asset('assets/logo-price.svg'),
                ),
              ),
            ),
            Positioned(
              right: 76,
              top: 40,
              child: Container(
                width: 30,
                height: 30,
                alignment: Alignment.center,
                decoration: BoxDecoration(
                  color: Colors.white,
                  shape: BoxShape.circle,
                  boxShadow: [BoxShadow(color: Colors.black.withValues(alpha: 0.12), blurRadius: 10, offset: const Offset(0, 4))],
                ),
                child: Text(emoji, style: const TextStyle(fontSize: 14)),
              ),
            ),
            const Positioned(right: 124, top: 14, child: _Cross(size: 11, color: AppColors.brand)),
            const Positioned(right: 12, top: 70, child: _Cross(size: 9, color: AppColors.ink)),
            const Positioned(right: 118, top: 56, child: _Dots()),
            Padding(
              padding: const EdgeInsets.fromLTRB(20, 20, 20, 22),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Padding(
                    padding: const EdgeInsets.only(right: 140),
                    child: EyebrowPill(eyebrow),
                  ),
                  const SizedBox(height: 40),
                  Text.rich(
                    TextSpan(
                      children: [
                        TextSpan(text: title),
                        if (highlight.isNotEmpty) ...[
                          const TextSpan(text: ' '),
                          TextSpan(text: highlight, style: const TextStyle(color: AppColors.brand)),
                        ],
                      ],
                    ),
                    style: const TextStyle(
                      fontSize: 27,
                      height: 1.1,
                      fontWeight: FontWeight.w900,
                      color: AppColors.ink,
                      letterSpacing: -0.6,
                    ),
                  ),
                  if (subtitle != null) ...[
                    const SizedBox(height: 8),
                    DefaultTextStyle.merge(
                      style: const TextStyle(fontSize: 13, height: 1.45, color: AppColors.gray),
                      child: subtitle!,
                    ),
                  ],
                  if (children.isNotEmpty) ...[
                    const SizedBox(height: 14),
                    ...children,
                  ],
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  static Widget _circle(double size, {Color? color, Color? border}) => Container(
        width: size,
        height: size,
        decoration: BoxDecoration(
          color: color,
          shape: BoxShape.circle,
          border: border == null ? null : Border.all(color: border, width: 2),
        ),
      );
}

class _Cross extends StatelessWidget {
  final double size;
  final Color color;

  const _Cross({required this.size, required this.color});

  @override
  Widget build(BuildContext context) {
    return Opacity(
      opacity: 0.7,
      child: Icon(Icons.close_rounded, size: size + 6, color: color, weight: 900),
    );
  }
}

class _Dots extends StatelessWidget {
  const _Dots();

  @override
  Widget build(BuildContext context) {
    Widget dot() => Container(
          width: 4,
          height: 4,
          margin: const EdgeInsets.all(3),
          decoration: BoxDecoration(color: AppColors.brand.withValues(alpha: 0.45), shape: BoxShape.circle),
        );
    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        Row(mainAxisSize: MainAxisSize.min, children: [dot(), dot(), dot()]),
        Row(mainAxisSize: MainAxisSize.min, children: [dot(), dot(), dot()]),
      ],
    );
  }
}

/// A rounded square with an icon, as used in section headers.
class IconTile extends StatelessWidget {
  final IconData icon;
  final double size;
  final Color color;
  final Color background;

  const IconTile(this.icon, {super.key, this.size = 40, this.color = AppColors.brand, this.background = AppColors.accentBg});

  @override
  Widget build(BuildContext context) {
    return Container(
      width: size,
      height: size,
      decoration: BoxDecoration(color: background, borderRadius: BorderRadius.circular(size * 0.32)),
      child: Icon(icon, color: color, size: size * 0.5),
    );
  }
}

/// Icon tile + title (+ subtitle) + optional trailing widget.
class SectionHeader extends StatelessWidget {
  final IconData icon;
  final String title;
  final String? subtitle;
  final Widget? trailing;

  const SectionHeader({super.key, required this.icon, required this.title, this.subtitle, this.trailing});

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        IconTile(icon),
        const SizedBox(width: 12),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(title, style: const TextStyle(fontSize: 15, fontWeight: FontWeight.w800, color: AppColors.ink)),
              if (subtitle != null) ...[
                const SizedBox(height: 2),
                Text(subtitle!, style: const TextStyle(fontSize: 12, height: 1.35, color: AppColors.gray)),
              ],
            ],
          ),
        ),
        if (trailing != null) trailing!,
      ],
    );
  }
}

/// A small rounded pill: "3 itens", "menor preço".
class AppChip extends StatelessWidget {
  final String text;
  final Color color;
  final Color? background;
  final IconData? icon;

  const AppChip(this.text, {super.key, this.color = AppColors.brand, this.background, this.icon});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
      decoration: BoxDecoration(
        color: background ?? color.withValues(alpha: 0.12),
        borderRadius: BorderRadius.circular(999),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (icon != null) ...[
            Icon(icon, size: 12, color: color),
            const SizedBox(width: 4),
          ],
          Flexible(
            child: Text(
              text,
              overflow: TextOverflow.ellipsis,
              style: TextStyle(fontSize: 11, fontWeight: FontWeight.w700, color: color),
            ),
          ),
        ],
      ),
    );
  }
}

/// A white pill with a colored dot: "4 mercados", "no mapa".
class StatChip extends StatelessWidget {
  final String text;
  final Color dot;

  const StatChip(this.text, {super.key, this.dot = AppColors.brand});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 11, vertical: 6),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(999),
        border: Border.all(color: AppColors.border),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(width: 7, height: 7, decoration: BoxDecoration(color: dot, shape: BoxShape.circle)),
          const SizedBox(width: 6),
          Flexible(
            child: Text(
              text,
              overflow: TextOverflow.ellipsis,
              style: const TextStyle(fontSize: 11, fontWeight: FontWeight.w700, color: AppColors.gray),
            ),
          ),
        ],
      ),
    );
  }
}

/// A centered illustration-style empty state: icon in a blue disc, title,
/// message and an optional action.
class EmptyState extends StatelessWidget {
  final IconData icon;
  final String title;
  final String? message;
  final Widget? action;

  const EmptyState({super.key, required this.icon, required this.title, this.message, this.action});

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 32, vertical: 24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Stack(
              alignment: Alignment.center,
              children: [
                Container(
                  width: 112,
                  height: 112,
                  decoration: BoxDecoration(shape: BoxShape.circle, border: Border.all(color: AppColors.heroRing, width: 2)),
                ),
                Container(
                  width: 84,
                  height: 84,
                  decoration: const BoxDecoration(color: AppColors.accentBg, shape: BoxShape.circle),
                  child: Icon(icon, size: 38, color: AppColors.brand),
                ),
              ],
            ),
            const SizedBox(height: 18),
            Text(
              title,
              textAlign: TextAlign.center,
              style: const TextStyle(fontSize: 17, fontWeight: FontWeight.w900, color: AppColors.ink),
            ),
            if (message != null) ...[
              const SizedBox(height: 6),
              Text(
                message!,
                textAlign: TextAlign.center,
                style: const TextStyle(fontSize: 13, height: 1.45, color: AppColors.gray),
              ),
            ],
            if (action != null) ...[
              const SizedBox(height: 18),
              action!,
            ],
          ],
        ),
      ),
    );
  }
}
