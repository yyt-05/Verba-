import 'package:flutter/material.dart';
import '../../models/subtitle_entry.dart';
import '../../theme/verba_theme.dart';
import '../../utils/translation_correction_diff.dart';
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
    final translationSpans = _translationSpans(recent);
    final hasContent = translation.isNotEmpty || original.isNotEmpty;

    return LayoutBuilder(
      builder: (context, constraints) {
        final maxWidth = constraints.hasBoundedWidth
            ? constraints.maxWidth - 24
            : 720.0;
        final width = (560.0 * fontScale)
            .clamp(360.0, maxWidth.clamp(360.0, 760.0))
            .toDouble();
        final height = (218.0 * fontScale).clamp(190.0, 300.0).toDouble();

        return GestureDetector(
          onTap: onTap,
          child: SizedBox(
            width: width,
            height: height,
            child: GlassSurface(
              radius: 12,
              opacity: 0.46,
              borderColor: VerbaColors.inkWhite,
              padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
              child: SizedBox.expand(
                child: hasContent
                    ? _ParagraphView(
                        original: original,
                        translation: translation,
                        translationSpans: translationSpans,
                        fontScale: fontScale,
                      )
                    : const Center(
                        child: Text(
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
    final buffer = StringBuffer();
    String? previousSpeaker;
    for (final entry in entries) {
      final text = entry.original.trim();
      if (text.isEmpty) continue;
      _writeSpeakerSeparatedText(
        buffer: buffer,
        text: text,
        separator: ' ',
        previousSpeaker: previousSpeaker,
        currentSpeaker: entry.speaker,
      );
      previousSpeaker = entry.speaker;
    }
    return buffer.toString();
  }

  String _joinTranslation(List<SubtitleEntry> entries) {
    final buffer = StringBuffer();
    String? previousSpeaker;
    for (final entry in entries) {
      final text = entry.translation.trim();
      if (text.isEmpty) continue;
      _writeSpeakerSeparatedText(
        buffer: buffer,
        text: text,
        separator: '',
        previousSpeaker: previousSpeaker,
        currentSpeaker: entry.speaker,
      );
      previousSpeaker = entry.speaker;
    }
    return buffer.toString();
  }

  List<TranslationCorrectionPart>? _translationSpans(
    List<SubtitleEntry> entries,
  ) {
    if (!correctionPreview) return null;
    final parts = <TranslationCorrectionPart>[];
    String? previousSpeaker;
    for (final entry in entries) {
      final translation = entry.translation.trim();
      final oldTranslation = entry.oldTranslation?.trim();
      if (translation.isNotEmpty &&
          _speakerChanged(previousSpeaker, entry.speaker)) {
        parts.add(
          const TranslationCorrectionPart(
            '\n',
            TranslationCorrectionPartKind.unchanged,
          ),
        );
      }
      final showCorrection =
          entry.isCorrected &&
          oldTranslation != null &&
          oldTranslation.isNotEmpty &&
          translation.isNotEmpty;
      if (showCorrection) {
        parts.addAll(
          buildTranslationCorrectionParts(
            oldText: oldTranslation,
            newText: translation,
          ),
        );
      } else if (translation.isNotEmpty) {
        parts.add(
          TranslationCorrectionPart(
            translation,
            TranslationCorrectionPartKind.unchanged,
          ),
        );
      }
      if (translation.isNotEmpty) previousSpeaker = entry.speaker;
    }
    return parts.isEmpty ? null : parts;
  }

  void _writeSpeakerSeparatedText({
    required StringBuffer buffer,
    required String text,
    required String separator,
    required String? previousSpeaker,
    required String currentSpeaker,
  }) {
    if (buffer.isNotEmpty) {
      buffer.write(
        _speakerChanged(previousSpeaker, currentSpeaker) ? '\n' : separator,
      );
    }
    buffer.write(text);
  }

  bool _speakerChanged(String? previousSpeaker, String currentSpeaker) {
    return previousSpeaker != null &&
        previousSpeaker.isNotEmpty &&
        currentSpeaker.isNotEmpty &&
        previousSpeaker != currentSpeaker;
  }
}

class _ParagraphView extends StatelessWidget {
  final String original;
  final String translation;
  final List<TranslationCorrectionPart>? translationSpans;
  final double fontScale;

  const _ParagraphView({
    required this.original,
    required this.translation,
    required this.translationSpans,
    required this.fontScale,
  });

  @override
  Widget build(BuildContext context) {
    return Column(
      mainAxisAlignment: MainAxisAlignment.end,
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        if (original.isNotEmpty)
          Flexible(
            flex: 3,
            child: Align(
              alignment: Alignment.bottomLeft,
              child: Text(
                original,
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
                style: _shadowStyle(
                  color: VerbaColors.inkWhite.withValues(alpha: 0.64),
                  size: 13 * fontScale,
                  weight: FontWeight.w600,
                ),
              ),
            ),
          ),
        if (original.isNotEmpty && translation.isNotEmpty)
          const SizedBox(height: 6),
        if (translation.isNotEmpty)
          Flexible(
            flex: 5,
            child: Align(
              alignment: Alignment.bottomLeft,
              child: _TranslationText(
                translation: translation,
                spans: translationSpans,
                fontScale: fontScale,
              ),
            ),
          ),
      ],
    );
  }
}

class _TranslationText extends StatelessWidget {
  final String translation;
  final List<TranslationCorrectionPart>? spans;
  final double fontScale;

  const _TranslationText({
    required this.translation,
    required this.spans,
    required this.fontScale,
  });

  @override
  Widget build(BuildContext context) {
    final baseStyle = _shadowStyle(
      color: VerbaColors.inkWhite,
      size: 21 * fontScale,
      weight: FontWeight.w900,
    );

    if (spans == null) {
      return Text(
        translation,
        maxLines: 3,
        overflow: TextOverflow.ellipsis,
        style: baseStyle,
      );
    }

    return RichText(
      maxLines: 3,
      overflow: TextOverflow.ellipsis,
      text: TextSpan(
        style: baseStyle,
        children: spans!.map((part) {
          if (part.kind == TranslationCorrectionPartKind.oldText) {
            return TextSpan(
              text: part.text,
              style: baseStyle.copyWith(
                color: VerbaColors.mutedGray.withValues(alpha: 0.76),
                decoration: TextDecoration.lineThrough,
                decorationColor: VerbaColors.mutedGray.withValues(alpha: 0.7),
                decorationThickness: 2,
              ),
            );
          }
          if (part.kind == TranslationCorrectionPartKind.newText) {
            return TextSpan(
              text: part.text,
              style: baseStyle.copyWith(color: VerbaColors.accentYellow),
            );
          }
          return TextSpan(text: part.text);
        }).toList(),
      ),
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
