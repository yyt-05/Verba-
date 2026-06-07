import 'package:flutter/material.dart';
import '../../models/subtitle_entry.dart';
import '../../theme/verba_theme.dart';
import 'glass_surface.dart';

class LyricSubtitleCard extends StatelessWidget {
  final List<SubtitleEntry> subtitles;
  final bool correctionPreview;
  final double fontScale;
  final VoidCallback onTap;

  const LyricSubtitleCard({
    super.key,
    required this.subtitles,
    required this.correctionPreview,
    required this.fontScale,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    final recent = _recentParagraph(subtitles);
    final original = _joinOriginal(recent);
    final translation = _joinTranslation(recent);
    final hasContent = translation.isNotEmpty || original.isNotEmpty;

    return LayoutBuilder(
      builder: (context, constraints) {
        final maxWidth = constraints.hasBoundedWidth
            ? constraints.maxWidth - 24
            : 720.0;
        final width = (560.0 * fontScale)
            .clamp(360.0, maxWidth.clamp(360.0, 760.0))
            .toDouble();

        return GestureDetector(
          onTap: onTap,
          child: AnimatedContainer(
            duration: const Duration(milliseconds: 220),
            width: width,
            child: GlassSurface(
              radius: 12,
              opacity: 0.46,
              borderColor: VerbaColors.inkWhite,
              padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
              child: hasContent
                  ? _ParagraphView(
                      original: original,
                      translation: translation,
                      fontScale: fontScale,
                    )
                  : const Text(
                      '正在等待英文音频...',
                      textAlign: TextAlign.center,
                      style: TextStyle(
                        color: VerbaColors.mutedGray,
                        fontSize: 14,
                        fontWeight: FontWeight.w700,
                        decoration: TextDecoration.none,
                      ),
                    ),
            ),
          ),
        );
      },
    );
  }

  List<SubtitleEntry> _recentParagraph(List<SubtitleEntry> entries) {
    if (entries.isEmpty) return const [];
    final start = entries.length > 5 ? entries.length - 5 : 0;
    return entries.sublist(start);
  }

  String _joinOriginal(List<SubtitleEntry> entries) {
    return entries
        .map((entry) => entry.original.trim())
        .where((text) => text.isNotEmpty)
        .join(' ');
  }

  String _joinTranslation(List<SubtitleEntry> entries) {
    return entries
        .map((entry) => entry.translation.trim())
        .where((text) => text.isNotEmpty)
        .join('');
  }
}

class _ParagraphView extends StatelessWidget {
  final String original;
  final String translation;
  final double fontScale;

  const _ParagraphView({
    required this.original,
    required this.translation,
    required this.fontScale,
  });

  @override
  Widget build(BuildContext context) {
    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        if (original.isNotEmpty)
          Text(
            original,
            maxLines: 4,
            overflow: TextOverflow.ellipsis,
            style: _shadowStyle(
              color: VerbaColors.inkWhite.withValues(alpha: 0.64),
              size: 13 * fontScale,
              weight: FontWeight.w600,
            ),
          ),
        if (original.isNotEmpty && translation.isNotEmpty)
          const SizedBox(height: 8),
        if (translation.isNotEmpty)
          Text(
            translation,
            maxLines: 5,
            overflow: TextOverflow.ellipsis,
            style: _shadowStyle(
              color: VerbaColors.inkWhite,
              size: 21 * fontScale,
              weight: FontWeight.w900,
            ),
          ),
      ],
    );
  }
}

TextStyle _shadowStyle({
  required Color color,
  required double size,
  required FontWeight weight,
}) {
  return TextStyle(
    color: color,
    fontSize: size,
    fontWeight: weight,
    height: 1.35,
    decoration: TextDecoration.none,
    shadows: [
      Shadow(
        color: Colors.black.withValues(alpha: 0.86),
        blurRadius: 8,
        offset: const Offset(0, 2),
      ),
      Shadow(color: Colors.black.withValues(alpha: 0.48), blurRadius: 14),
    ],
  );
}
