import 'package:flutter/material.dart';
import '../../models/subtitle_entry.dart';
import '../../theme/verba_theme.dart';
import '../../utils/translation_correction_diff.dart';
import 'floating_icon.dart';
import 'glass_surface.dart';

class SlidingSubtitlePane extends StatelessWidget {
  final List<SubtitleEntry> subtitles;
  final VoidCallback onCollapse;

  const SlidingSubtitlePane({
    super.key,
    required this.subtitles,
    required this.onCollapse,
  });

  @override
  Widget build(BuildContext context) {
    final recent = subtitles.length > 12
        ? subtitles.sublist(subtitles.length - 12)
        : subtitles;

    return AnimatedSwitcher(
      duration: const Duration(milliseconds: 220),
      child: GlassSurface(
        radius: 16,
        opacity: 0.78,
        borderColor: VerbaColors.inkWhite,
        padding: EdgeInsets.zero,
        child: SizedBox(
          width: 390,
          height: 286,
          child: Column(
            children: [
              Padding(
                padding: const EdgeInsets.fromLTRB(16, 12, 12, 8),
                child: Row(
                  children: [
                    const Text(
                      '最近字幕',
                      style: TextStyle(
                        color: VerbaColors.inkWhite,
                        fontSize: 16,
                        fontWeight: FontWeight.w900,
                        decoration: TextDecoration.none,
                      ),
                    ),
                    const Spacer(),
                    Text(
                      '${recent.length} 条',
                      style: const TextStyle(
                        color: VerbaColors.mutedGray,
                        fontSize: 12,
                        fontWeight: FontWeight.w700,
                        decoration: TextDecoration.none,
                      ),
                    ),
                    const SizedBox(width: 8),
                    GestureDetector(
                      onTap: onCollapse,
                      child: const FloatingIcon(
                        name: 'verba-collapse',
                        size: 28,
                      ),
                    ),
                  ],
                ),
              ),
              Divider(color: Colors.white.withValues(alpha: 0.08), height: 1),
              Expanded(
                child: recent.isEmpty
                    ? const Center(
                        child: Text(
                          '暂无字幕',
                          style: TextStyle(
                            color: VerbaColors.mutedGray,
                            fontWeight: FontWeight.w700,
                            decoration: TextDecoration.none,
                          ),
                        ),
                      )
                    : ListView.builder(
                        padding: const EdgeInsets.symmetric(
                          horizontal: 14,
                          vertical: 10,
                        ),
                        itemCount: recent.length,
                        itemBuilder: (context, index) =>
                            _SubtitleRow(entry: recent[index]),
                      ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _SubtitleRow extends StatefulWidget {
  final SubtitleEntry entry;

  const _SubtitleRow({required this.entry});

  @override
  State<_SubtitleRow> createState() => _SubtitleRowState();
}

class _SubtitleRowState extends State<_SubtitleRow>
    with SingleTickerProviderStateMixin {
  AnimationController? _flashController;

  @override
  void didUpdateWidget(covariant _SubtitleRow oldWidget) {
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
          setState(() {});
        }
      });
  }

  @override
  void dispose() {
    _flashController?.dispose();
    super.dispose();
  }

  Color _flashBg() {
    if (_flashController == null || !_flashController!.isAnimating) {
      return Colors.transparent;
    }
    final t = Curves.easeOut.transform(_flashController!.value);
    return Color.lerp(
      VerbaColors.accentYellow.withValues(alpha: 0.18),
      Colors.transparent,
      t,
    )!;
  }

  Color _newTextColor() {
    if (_flashController == null || !_flashController!.isAnimating) {
      return VerbaColors.inkWhite;
    }
    final t = Curves.easeOut.transform(_flashController!.value);
    return Color.lerp(VerbaColors.accentYellow, VerbaColors.inkWhite, t)!;
  }

  @override
  Widget build(BuildContext context) {
    final entry = widget.entry;
    final showCorrection =
        entry.isCorrected &&
        entry.oldTranslation != null &&
        entry.oldTranslation!.isNotEmpty;

    return Container(
      margin: const EdgeInsets.only(bottom: 10),
      padding: const EdgeInsets.fromLTRB(10, 6, 8, 7),
      decoration: BoxDecoration(
        color: showCorrection
            ? _flashBg()
            : Colors.white.withValues(alpha: 0.02),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (entry.speaker.isNotEmpty)
            Padding(
              padding: const EdgeInsets.only(bottom: 3),
              child: _MiniSpeakerBadge(label: entry.speaker),
            ),
          Text(
            entry.original,
            maxLines: 2,
            overflow: TextOverflow.ellipsis,
            style: TextStyle(
              color: VerbaColors.inkWhite.withValues(alpha: 0.58),
              fontSize: 12,
              fontWeight: FontWeight.w600,
              decoration: TextDecoration.none,
            ),
          ),
          SizedBox(height: showCorrection ? 2 : 3),
          showCorrection
              ? _CorrectionTranslationText(
                  oldText: entry.oldTranslation!,
                  newText: entry.translation,
                  newTextColor: _newTextColor(),
                )
              : Text(
                  entry.translation,
                  maxLines: 3,
                  overflow: TextOverflow.ellipsis,
                  style: const TextStyle(
                    color: VerbaColors.inkWhite,
                    fontSize: 15,
                    fontWeight: FontWeight.w800,
                    height: 1.35,
                    decoration: TextDecoration.none,
                  ),
                ),
        ],
      ),
    );
  }
}

class _CorrectionTranslationText extends StatelessWidget {
  final String oldText;
  final String newText;
  final Color newTextColor;

  const _CorrectionTranslationText({
    required this.oldText,
    required this.newText,
    required this.newTextColor,
  });

  @override
  Widget build(BuildContext context) {
    const baseStyle = TextStyle(
      color: VerbaColors.inkWhite,
      fontSize: 15,
      fontWeight: FontWeight.w800,
      height: 1.35,
      decoration: TextDecoration.none,
    );

    return RichText(
      maxLines: 3,
      overflow: TextOverflow.ellipsis,
      text: TextSpan(
        style: baseStyle,
        children:
            buildTranslationCorrectionParts(
              oldText: oldText,
              newText: newText,
            ).map((part) {
              if (part.kind == TranslationCorrectionPartKind.oldText) {
                return TextSpan(
                  text: part.text,
                  style: baseStyle.copyWith(
                    color: VerbaColors.mutedGray.withValues(alpha: 0.6),
                    decoration: TextDecoration.lineThrough,
                    decorationColor: VerbaColors.mutedGray.withValues(
                      alpha: 0.5,
                    ),
                    decorationThickness: 1.5,
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

Color _speakerColor(String label) {
  if (label.isEmpty) return Colors.grey;
  final c = label.codeUnitAt(0);
  const palette = [
    Color(0xFF4A9EFF),
    Color(0xFF12C76A),
    Color(0xFFFF8C42),
    Color(0xFFC084FC),
    Color(0xFFF472B6),
  ];
  final idx = (c - 65).clamp(0, palette.length - 1);
  return palette[idx];
}

class _MiniSpeakerBadge extends StatelessWidget {
  final String label;
  const _MiniSpeakerBadge({required this.label});

  @override
  Widget build(BuildContext context) {
    final color = _speakerColor(label);
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Container(
          width: 6,
          height: 6,
          decoration: BoxDecoration(color: color, shape: BoxShape.circle),
        ),
        const SizedBox(width: 3),
        Text(
          '说话人 $label',
          style: TextStyle(
            color: color.withValues(alpha: 0.8),
            fontSize: 10,
            fontWeight: FontWeight.w700,
            decoration: TextDecoration.none,
          ),
        ),
      ],
    );
  }
}
