import 'package:flutter/material.dart';
import '../../theme/verba_theme.dart';
import 'floating_icon.dart';
import 'glass_surface.dart';

class MiniFloatingPill extends StatelessWidget {
  final VoidCallback onStart;

  const MiniFloatingPill({super.key, required this.onStart});

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: onStart,
      child: MouseRegion(
        cursor: SystemMouseCursors.click,
        child: GlassSurface(
          radius: 28,
          opacity: 0.18,
          padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: const [
              FloatingIcon(name: 'verba-start', size: 34),
              SizedBox(width: 10),
              Text(
                'Verba',
                style: TextStyle(
                  color: VerbaColors.inkWhite,
                  fontSize: 18,
                  fontWeight: FontWeight.w800,
                ),
              ),
              SizedBox(width: 8),
            ],
          ),
        ),
      ),
    );
  }
}
