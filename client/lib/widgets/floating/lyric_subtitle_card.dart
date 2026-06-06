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
    final current = subtitles.isNotEmpty ? subtitles.last : null;
    final previous = subtitles.length > 1
        ? subtitles[subtitles.length - 2]
        : null;
    final corrected = _latestCorrected(subtitles);
    final showCorrection = correctionPreview && corrected != null;

    return LayoutBuilder(
      builder: (context, constraints) {
        final maxWidth = constraints.hasBoundedWidth
            ? constraints.maxWidth - 24
            : 520.0;
        final targetWidth = (showCorrection ? 480.0 : 420.0) * fontScale;
        final width = targetWidth
            .clamp(320.0, maxWidth.clamp(320.0, 560.0))
            .toDouble();

        return GestureDetector(
          onTap: onTap,
          child: AnimatedContainer(
            duration: const Duration(milliseconds: 220),
            width: width,
            child: GlassSurface(
              radius: 12,
              opacity: showCorrection ? 0.58 : 0.48,
              borderColor: VerbaColors.inkWhite,
              padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
              child: showCorrection
                  ? _CorrectionPreview(
                      entry: corrected,
                      subtitles: subtitles,
                      fontScale: fontScale,
                    )
                  : _LyricPair(
                      previous: previous,
                      current: current,
                      fontScale: fontScale,
                    ),
            ),
          ),
        );
      },
    );
  }

  SubtitleEntry? _latestCorrected(List<SubtitleEntry> entries) {
    for (var i = entries.length - 1; i >= 0; i--) {
      if (entries[i].isCorrected) return entries[i];
    }
    return null;
  }
}

class _LyricPair extends StatelessWidget {
  final SubtitleEntry? previous;
  final SubtitleEntry? current;
  final double fontScale;

  const _LyricPair({
    required this.previous,
    required this.current,
    required this.fontScale,
  });

  @override
  Widget build(BuildContext context) {
    if (current == null) {
      return const Text(
        '正在等待英文音频...',
        textAlign: TextAlign.center,
        style: TextStyle(
          color: VerbaColors.mutedGray,
          fontSize: 14,
          fontWeight: FontWeight.w700,
        ),
      );
    }

    final currentAlpha = current?.status == SubtitleStatus.draft ? 0.58 : 1.0;
    final sourceAlpha = current?.status == SubtitleStatus.draft ? 0.48 : 0.76;

    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        if (previous != null)
          Text(
            previous!.translation,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            textAlign: TextAlign.center,
            style: _shadowStyle(
              color: VerbaColors.inkWhite.withValues(alpha: 0.62),
              size: 13 * fontScale,
              weight: FontWeight.w600,
            ),
          ),
        if (previous != null) const SizedBox(height: 4),
        Text(
          current!.original,
          maxLines: 2,
          overflow: TextOverflow.ellipsis,
          textAlign: TextAlign.center,
          style: _shadowStyle(
            color: VerbaColors.inkWhite.withValues(alpha: sourceAlpha),
            size: 15 * fontScale,
            weight: FontWeight.w600,
          ),
        ),
        const SizedBox(height: 4),
        Text(
          current!.translation,
          maxLines: 2,
          overflow: TextOverflow.ellipsis,
          textAlign: TextAlign.center,
          style: _shadowStyle(
            color: VerbaColors.inkWhite.withValues(alpha: currentAlpha),
            size: 23 * fontScale,
            weight: FontWeight.w900,
          ),
        ),
      ],
    );
  }
}

class _CorrectionPreview extends StatelessWidget {
  final SubtitleEntry entry;
  final List<SubtitleEntry> subtitles;
  final double fontScale;

  const _CorrectionPreview({
    required this.entry,
    required this.subtitles,
    required this.fontScale,
  });

  @override
  Widget build(BuildContext context) {
    final index = subtitles.indexWhere((e) => e.segmentId == entry.segmentId);
    final previous = index > 0 ? subtitles[index - 1].translation : null;
    final next = index >= 0 && index < subtitles.length - 1
        ? subtitles[index + 1].translation
        : null;

    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        if (previous != null)
          Text(
            '上一句：$previous',
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: const TextStyle(
              color: VerbaColors.mutedGray,
              fontSize: 12,
              fontWeight: FontWeight.w700,
            ),
          ),
        const SizedBox(height: 8),
        Text(
          entry.original,
          maxLines: 2,
          overflow: TextOverflow.ellipsis,
          style: _shadowStyle(
            color: VerbaColors.inkWhite,
            size: 13 * fontScale,
            weight: FontWeight.w700,
          ),
        ),
        const SizedBox(height: 6),
        Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Container(
              width: 3,
              height: 48,
              margin: const EdgeInsets.only(top: 3, right: 10),
              decoration: BoxDecoration(
                color: VerbaColors.accentYellow,
                borderRadius: BorderRadius.circular(2),
              ),
            ),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Wrap(
                    crossAxisAlignment: WrapCrossAlignment.center,
                    spacing: 8,
                    runSpacing: 4,
                    children: [
                      Text(
                        entry.translation,
                        maxLines: 2,
                        overflow: TextOverflow.ellipsis,
                        style: _shadowStyle(
                          color: VerbaColors.inkWhite,
                          size: 22 * fontScale,
                          weight: FontWeight.w900,
                        ),
                      ),
                      const _CorrectionBadge(),
                    ],
                  ),
                  if (entry.oldTranslation != null)
                    Text(
                      '原译：${entry.oldTranslation}',
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: TextStyle(
                        color: VerbaColors.mutedGray.withValues(alpha: 0.74),
                        fontSize: 12,
                        fontWeight: FontWeight.w700,
                      ),
                    ),
                ],
              ),
            ),
          ],
        ),
        if (next != null) ...[
          const SizedBox(height: 8),
          Text(
            '下一句：$next',
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: const TextStyle(
              color: VerbaColors.mutedGray,
              fontSize: 12,
              fontWeight: FontWeight.w700,
            ),
          ),
        ],
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
    height: 1.25,
    shadows: [
      Shadow(
        color: Colors.black.withValues(alpha: 0.9),
        blurRadius: 8,
        offset: const Offset(0, 2),
      ),
      Shadow(color: Colors.black.withValues(alpha: 0.58), blurRadius: 14),
    ],
  );
}

class _CorrectionBadge extends StatelessWidget {
  const _CorrectionBadge();

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 7, vertical: 3),
      decoration: BoxDecoration(
        color: VerbaColors.accentYellow.withValues(alpha: 0.16),
        borderRadius: BorderRadius.circular(10),
        border: Border.all(
          color: VerbaColors.accentYellow.withValues(alpha: 0.34),
        ),
      ),
      child: const Text(
        '已修正',
        style: TextStyle(
          color: VerbaColors.accentYellow,
          fontSize: 11,
          fontWeight: FontWeight.w900,
        ),
      ),
    );
  }
}
