import 'dart:ui';

import 'package:flutter/material.dart';
import '../../theme/verba_theme.dart';

class GlassSurface extends StatelessWidget {
  final Widget child;
  final EdgeInsetsGeometry padding;
  final double radius;
  final double opacity;
  final Color borderColor;

  const GlassSurface({
    super.key,
    required this.child,
    this.padding = const EdgeInsets.all(12),
    this.radius = 12,
    this.opacity = 0.16,
    this.borderColor = VerbaColors.textBlue,
  });

  @override
  Widget build(BuildContext context) {
    return ClipRRect(
      borderRadius: BorderRadius.circular(radius),
      child: BackdropFilter(
        filter: ImageFilter.blur(sigmaX: 10, sigmaY: 10),
        child: Container(
          padding: padding,
          decoration: BoxDecoration(
            color: const Color(0xFF080B12).withValues(alpha: opacity),
            borderRadius: BorderRadius.circular(radius),
            border: Border.all(color: borderColor.withValues(alpha: 0.16)),
            boxShadow: [
              BoxShadow(
                color: Colors.black.withValues(alpha: 0.46),
                blurRadius: 20,
                offset: const Offset(0, 10),
              ),
              BoxShadow(
                color: VerbaColors.brandBlue.withValues(alpha: 0.05),
                blurRadius: 16,
              ),
            ],
          ),
          child: child,
        ),
      ),
    );
  }
}
