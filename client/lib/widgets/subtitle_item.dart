import 'package:flutter/material.dart';
import '../models/subtitle_entry.dart';
import '../theme/verba_theme.dart';
import '../utils/translation_correction_diff.dart';

class SubtitleItem extends StatefulWidget {
  final SubtitleEntry entry;
  final bool isNewest;

  const SubtitleItem({super.key, required this.entry, this.isNewest = false});

  @override
  State<SubtitleItem> createState() => _SubtitleItemState();
}

class _SubtitleItemState extends State<SubtitleItem>
    with SingleTickerProviderStateMixin {
  AnimationController? _flashController;

  @override
  void didUpdateWidget(covariant SubtitleItem oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (!oldWidget.entry.isCorrected && widget.entry.isCorrected) {
      _triggerFlash();
    }
  }

  void _triggerFlash() {
    _flashController?.dispose();
    _flashController = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 500),
    );
    _flashController!
      ..addListener(() => setState(() {}))
      ..forward().then((_) {
        if (mounted) {
          _flashController?.dispose();
          _flashController = null;
          setState(() {}); // settle to final look
        }
      });
  }

  @override
  void dispose() {
    _flashController?.dispose();
    super.dispose();
  }

  Color _flashColor() {
    if (_flashController == null || !_flashController!.isAnimating) {
      return Colors.transparent;
    }
    final t = _flashController!.value;
    return Color.lerp(
      VerbaColors.accentYellow.withValues(alpha: 0.18),
      Colors.transparent,
      Curves.easeOut.transform(t),
    )!;
  }

  Color _newTranslationColor() {
    if (_flashController == null || !_flashController!.isAnimating) {
      return VerbaColors.inkWhite;
    }
    final t = _flashController!.value;
    return Color.lerp(
      VerbaColors.accentYellow,
      VerbaColors.inkWhite,
      Curves.easeOut.transform(t),
    )!;
  }

  @override
  Widget build(BuildContext context) {
    final entry = widget.entry;
    final canCompareCorrection =
        entry.isCorrected &&
        entry.oldTranslation != null &&
        entry.oldTranslation!.isNotEmpty;
    final correctionParts = canCompareCorrection
        ? buildTranslationCorrectionParts(
            oldText: entry.oldTranslation!,
            newText: entry.translation,
          )
        : const <TranslationCorrectionPart>[];
    final showCorrection =
        canCompareCorrection &&
        shouldShowInlineCorrectionParts(correctionParts, entry.translation);

    return Container(
      margin: const EdgeInsets.only(bottom: 8),
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
      decoration: BoxDecoration(
        color: showCorrection
            ? _flashColor()
            : (widget.isNewest
                  ? Colors.white.withValues(alpha: 0.04)
                  : Colors.transparent),
        borderRadius: BorderRadius.circular(VerbaTheme.panelRadius),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (entry.speaker.isNotEmpty)
            Padding(
              padding: const EdgeInsets.only(bottom: 4),
              child: _SpeakerBadge(label: entry.speaker),
            ),
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
          SizedBox(height: showCorrection ? 2 : 3),
          showCorrection
              ? _CorrectionTranslationText(
                  parts: correctionParts,
                  newTextColor: _newTranslationColor(),
                )
              : Text(
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

class _CorrectionTranslationText extends StatelessWidget {
  final List<TranslationCorrectionPart> parts;
  final Color newTextColor;

  const _CorrectionTranslationText({
    required this.parts,
    required this.newTextColor,
  });

  @override
  Widget build(BuildContext context) {
    const baseStyle = TextStyle(
      color: VerbaColors.inkWhite,
      fontSize: 16,
      fontWeight: FontWeight.w700,
      height: 1.45,
      decoration: TextDecoration.none,
    );

    return RichText(
      text: TextSpan(
        style: baseStyle,
        children:
            parts.map((part) {
              if (part.kind == TranslationCorrectionPartKind.oldText) {
                return TextSpan(
                  text: part.text,
                  style: baseStyle.copyWith(
                    color: VerbaColors.mutedGray.withValues(alpha: 0.7),
                    decoration: TextDecoration.lineThrough,
                    decorationColor: VerbaColors.mutedGray.withValues(
                      alpha: 0.5,
                    ),
                    decorationThickness: 1.8,
                  ),
                );
              }
              if (part.kind == TranslationCorrectionPartKind.newText) {
                return TextSpan(
                  text: part.text,
                  style: baseStyle.copyWith(color: newTextColor),
                );
              }
              return TextSpan(text: part.text);
            }).toList(),
      ),
    );
  }
}

Color speakerColor(String label) {
  if (label.isEmpty) return Colors.grey;
  final c = label.codeUnitAt(0);
  const palette = [
    Color(0xFF4A9EFF), // A - blue
    Color(0xFF12C76A), // B - green
    Color(0xFFFF8C42), // C - orange
    Color(0xFFC084FC), // D - purple
    Color(0xFFF472B6), // E - pink
  ];
  final idx = (c - 65).clamp(0, palette.length - 1);
  return palette[idx];
}

class _SpeakerBadge extends StatelessWidget {
  final String label;
  const _SpeakerBadge({required this.label});

  @override
  Widget build(BuildContext context) {
    final color = speakerColor(label);
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Container(
          width: 8,
          height: 8,
          decoration: BoxDecoration(color: color, shape: BoxShape.circle),
        ),
        const SizedBox(width: 4),
        Text(
          '说话人 $label',
          style: TextStyle(
            color: color.withValues(alpha: 0.85),
            fontSize: 11,
            fontWeight: FontWeight.w700,
            decoration: TextDecoration.none,
          ),
        ),
      ],
    );
  }
}
