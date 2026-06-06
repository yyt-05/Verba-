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
    final corrected = _latestCorrected(subtitles);
    final showCorrection = correctionPreview && corrected != null;

    return LayoutBuilder(
      builder: (context, constraints) {
        final maxWidth = constraints.hasBoundedWidth
            ? constraints.maxWidth - 24
            : 720.0;
        final width = maxWidth.clamp(340.0, 920.0).toDouble();

        return GestureDetector(
          onTap: onTap,
          child: AnimatedContainer(
            duration: const Duration(milliseconds: 180),
            width: width,
            child: GlassSurface(
              radius: 10,
              opacity: showCorrection ? 0.42 : 0.34,
              borderColor: VerbaColors.textBlue,
              padding: const EdgeInsets.symmetric(horizontal: 18, vertical: 14),
              child: showCorrection
                  ? _CorrectionPreview(
                      entry: corrected,
                      subtitles: subtitles,
                      fontScale: fontScale,
                    )
                  : _ParagraphSubtitle(entries: recent, fontScale: fontScale),
            ),
          ),
        );
      },
    );
  }

  List<SubtitleEntry> _recentParagraph(List<SubtitleEntry> entries) {
    if (entries.isEmpty) return const [];
    final selected = <SubtitleEntry>[];
    var originalChars = 0;
    var translationChars = 0;

    for (var i = entries.length - 1; i >= 0 && selected.length < 4; i--) {
      final entry = entries[i];
      originalChars += entry.original.length;
      translationChars += entry.translation.length;
      selected.insert(0, entry);
      if (originalChars > 260 || translationChars > 180) break;
    }
    return selected;
  }

  SubtitleEntry? _latestCorrected(List<SubtitleEntry> entries) {
    for (var i = entries.length - 1; i >= 0; i--) {
      if (entries[i].isCorrected) return entries[i];
    }
    return null;
  }
}

class _ParagraphSubtitle extends StatelessWidget {
  final List<SubtitleEntry> entries;
  final double fontScale;

  const _ParagraphSubtitle({required this.entries, required this.fontScale});

  @override
  Widget build(BuildContext context) {
    if (entries.isEmpty) {
      return const Text(
        '正在等待英文音频...',
        textAlign: TextAlign.center,
        style: TextStyle(
          color: VerbaColors.mutedGray,
          fontSize: 14,
          fontWeight: FontWeight.w700,
          decoration: TextDecoration.none,
        ),
      );
    }

    final current = entries.last;
    final isDraft = current.status == SubtitleStatus.draft;
    final sourceAlpha = isDraft ? 0.42 : 0.68;
    final original = entries.map((e) => e.original).join(' ');

    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          original,
          textAlign: TextAlign.left,
          softWrap: true,
          style: _subtitleStyle(
            color: VerbaColors.inkWhite.withValues(alpha: sourceAlpha),
            size: 13 * fontScale,
            weight: FontWeight.w600,
          ),
        ),
        const SizedBox(height: 8),
        RichText(
          text: TextSpan(
            children: [
              for (final entry in entries)
                ..._translationSpans(
                  entry,
                  size: 21 * fontScale,
                  weight: FontWeight.w800,
                  isDraft: isDraft,
                ),
            ],
          ),
          softWrap: true,
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
          Text('上一句：$previous', softWrap: true, style: _metaStyle()),
        const SizedBox(height: 8),
        Text(
          entry.original,
          softWrap: true,
          style: _subtitleStyle(
            color: VerbaColors.inkWhite.withValues(alpha: 0.72),
            size: 13 * fontScale,
            weight: FontWeight.w600,
          ),
        ),
        const SizedBox(height: 6),
        RichText(
          text: TextSpan(
            children: _translationSpans(
              entry,
              size: 21 * fontScale,
              weight: FontWeight.w800,
              isDraft: false,
            ),
          ),
          softWrap: true,
        ),
        if (next != null) ...[
          const SizedBox(height: 8),
          Text('下一句：$next', softWrap: true, style: _metaStyle()),
        ],
      ],
    );
  }
}

List<TextSpan> _translationSpans(
  SubtitleEntry entry, {
  required double size,
  required FontWeight weight,
  required bool isDraft,
}) {
  final oldText = entry.oldTranslation?.trim();
  if (entry.isCorrected && oldText != null && oldText.isNotEmpty) {
    return [
      TextSpan(
        text: oldText,
        style: _subtitleStyle(
          color: VerbaColors.mutedGray.withValues(alpha: 0.72),
          size: size,
          weight: weight,
          decoration: TextDecoration.lineThrough,
        ),
      ),
      TextSpan(
        text: ' ${entry.translation}',
        style: _subtitleStyle(
          color: VerbaColors.accentYellow,
          size: size,
          weight: FontWeight.w900,
        ),
      ),
    ];
  }

  return [
    TextSpan(
      text: entry.translation,
      style: _subtitleStyle(
        color: VerbaColors.inkWhite.withValues(alpha: isDraft ? 0.62 : 1.0),
        size: size,
        weight: weight,
      ),
    ),
  ];
}

TextStyle _subtitleStyle({
  required Color color,
  required double size,
  required FontWeight weight,
  TextDecoration decoration = TextDecoration.none,
}) {
  return TextStyle(
    color: color,
    fontSize: size,
    fontWeight: weight,
    height: 1.28,
    decoration: decoration,
    decorationColor: color.withValues(alpha: 0.82),
    decorationThickness: 1.4,
    shadows: [
      Shadow(
        color: Colors.black.withValues(alpha: 0.68),
        blurRadius: 5,
        offset: const Offset(0, 2),
      ),
      Shadow(color: Colors.black.withValues(alpha: 0.28), blurRadius: 10),
    ],
  );
}

TextStyle _metaStyle() {
  return TextStyle(
    color: VerbaColors.mutedGray.withValues(alpha: 0.82),
    fontSize: 12,
    fontWeight: FontWeight.w700,
    height: 1.25,
    decoration: TextDecoration.none,
  );
}
