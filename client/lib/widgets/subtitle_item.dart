import 'package:flutter/material.dart';
import '../models/subtitle_entry.dart';
import '../theme/verba_theme.dart';

class SubtitleItem extends StatelessWidget {
  final SubtitleEntry entry;
  final bool isNewest;

  const SubtitleItem({super.key, required this.entry, this.isNewest = false});

  @override
  Widget build(BuildContext context) {
    return Container(
      margin: const EdgeInsets.only(bottom: 8),
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
      decoration: BoxDecoration(
        color: isNewest
            ? Colors.white.withValues(alpha: 0.04)
            : Colors.transparent,
        borderRadius: BorderRadius.circular(VerbaTheme.panelRadius),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            entry.original,
            style: TextStyle(
              color: Colors.white.withValues(alpha: 0.72),
              fontSize: 13,
              fontWeight: FontWeight.w600,
              height: 1.35,
              decoration: TextDecoration.none,
            ),
          ),
          const SizedBox(height: 3),
          Text(
            entry.translation,
            style: const TextStyle(
              color: VerbaColors.inkWhite,
              fontSize: 16,
              fontWeight: FontWeight.w700,
              height: 1.45,
              decoration: TextDecoration.none,
            ),
          ),
        ],
      ),
    );
  }
}
