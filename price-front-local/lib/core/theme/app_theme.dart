import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';

/// The Price palette, shared with the web panels: blue identity on a soft
/// blue-gray page, white rounded cards, pill buttons.
class AppColors {
  static const brand = Color(0xFF2A5CB9);
  static const brandDark = Color(0xFF1B3E82);
  static const brandBright = Color(0xFF3D72D6);
  static const accentBg = Color(0xFFE3ECFA);
  static const heroBg = Color(0xFFEEF3FC);
  static const heroRing = Color(0xFFC9D8F5);

  static const ink = Color(0xFF1C1C1E);
  static const gray = Color(0xFF5F6470);
  static const grayLight = Color(0xFF8E93A0);
  static const border = Color(0xFFE3E7EF);

  static const page = Color(0xFFF5F7FB);
  static const surface = Colors.white;
  static const field = Color(0xFFF7F9FC);

  static const red = Color(0xFFDC2626);
  static const redBg = Color(0xFFFDECEC);
  static const ok = Color(0xFF15803D);
  static const okBg = Color(0xFFDCFCE7);
  static const warn = Color(0xFFB45309);
  static const warnBg = Color(0xFFFEF3C7);

  static List<BoxShadow> get softShadow => [
        BoxShadow(color: const Color(0xFF1E3C78).withValues(alpha: 0.07), blurRadius: 24, offset: const Offset(0, 8)),
      ];
}

class AppTheme {
  static const Color primaryColor = AppColors.brand;
  static const Color secondaryColor = AppColors.ink;
  static const Color subtextColor = AppColors.gray;

  static ThemeData get lightTheme {
    final textTheme = GoogleFonts.alexandriaTextTheme();
    const pill = StadiumBorder();
    const buttonText = TextStyle(fontSize: 15, fontWeight: FontWeight.w700);

    return ThemeData(
      useMaterial3: true,
      colorScheme: ColorScheme.fromSeed(
        seedColor: primaryColor,
        primary: primaryColor,
        secondary: secondaryColor,
        surface: Colors.white,
      ),
      scaffoldBackgroundColor: AppColors.page,
      fontFamily: GoogleFonts.alexandria().fontFamily,
      textTheme: textTheme.copyWith(
        displayLarge: textTheme.displayLarge?.copyWith(color: secondaryColor),
        bodyLarge: textTheme.bodyLarge?.copyWith(color: secondaryColor),
        bodyMedium: textTheme.bodyMedium?.copyWith(color: subtextColor),
      ),
      appBarTheme: const AppBarTheme(
        backgroundColor: AppColors.page,
        surfaceTintColor: Colors.transparent,
        elevation: 0,
        iconTheme: IconThemeData(color: secondaryColor),
      ),
      elevatedButtonTheme: ElevatedButtonThemeData(
        style: ElevatedButton.styleFrom(
          backgroundColor: AppColors.brand,
          foregroundColor: Colors.white,
          disabledBackgroundColor: AppColors.accentBg,
          disabledForegroundColor: AppColors.grayLight,
          padding: const EdgeInsets.symmetric(vertical: 16, horizontal: 22),
          shape: pill,
          elevation: 0,
          textStyle: buttonText,
        ),
      ),
      outlinedButtonTheme: OutlinedButtonThemeData(
        style: OutlinedButton.styleFrom(
          foregroundColor: AppColors.brand,
          side: const BorderSide(color: AppColors.brand, width: 2),
          padding: const EdgeInsets.symmetric(vertical: 14, horizontal: 20),
          shape: pill,
          textStyle: buttonText,
        ),
      ),
      textButtonTheme: TextButtonThemeData(
        style: TextButton.styleFrom(
          foregroundColor: AppColors.brand,
          shape: pill,
          textStyle: const TextStyle(fontWeight: FontWeight.w700),
        ),
      ),
      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: AppColors.field,
        contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 16),
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(16),
          borderSide: const BorderSide(color: AppColors.border, width: 1.5),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(16),
          borderSide: const BorderSide(color: AppColors.border, width: 1.5),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(16),
          borderSide: const BorderSide(color: AppColors.brand, width: 2),
        ),
        prefixIconColor: AppColors.grayLight,
      ),
      snackBarTheme: SnackBarThemeData(
        behavior: SnackBarBehavior.floating,
        backgroundColor: AppColors.ink,
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
        actionTextColor: const Color(0xFF8FB2F5),
      ),
      dialogTheme: DialogThemeData(
        backgroundColor: Colors.white,
        surfaceTintColor: Colors.transparent,
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(24)),
        titleTextStyle: const TextStyle(fontSize: 19, fontWeight: FontWeight.w900, color: AppColors.ink),
      ),
      bottomSheetTheme: const BottomSheetThemeData(
        backgroundColor: Colors.white,
        surfaceTintColor: Colors.transparent,
        dragHandleColor: AppColors.border,
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.vertical(top: Radius.circular(28))),
      ),
      sliderTheme: const SliderThemeData(
        activeTrackColor: AppColors.brand,
        inactiveTrackColor: AppColors.accentBg,
        thumbColor: AppColors.brand,
        trackHeight: 6,
      ),
      progressIndicatorTheme: const ProgressIndicatorThemeData(color: AppColors.brand),
      navigationBarTheme: NavigationBarThemeData(
        backgroundColor: Colors.white,
        surfaceTintColor: Colors.transparent,
        indicatorColor: AppColors.accentBg,
        height: 68,
        labelTextStyle: WidgetStateProperty.resolveWith(
          (s) => TextStyle(
            fontSize: 11,
            fontWeight: s.contains(WidgetState.selected) ? FontWeight.w800 : FontWeight.w500,
            color: s.contains(WidgetState.selected) ? AppColors.brand : AppColors.grayLight,
          ),
        ),
        iconTheme: WidgetStateProperty.resolveWith(
          (s) => IconThemeData(color: s.contains(WidgetState.selected) ? AppColors.brand : AppColors.grayLight),
        ),
      ),
    );
  }
}
