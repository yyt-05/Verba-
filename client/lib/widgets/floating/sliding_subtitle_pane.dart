import 'package:flutter/material.dart';
import '../../models/subtitle_entry.dart';
import '../../theme/verba_theme.dart';
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
          width: 430,
          height: 330,
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

class _SubtitleRow extends StatelessWidget {
  final SubtitleEntry entry;

  const _SubtitleRow({required this.entry});

  @override
  Widget build(BuildContext context) {
    return Container(
      margin: const EdgeInsets.only(bottom: 12),
      padding: const EdgeInsets.fromLTRB(10, 7, 8, 8),
      decoration: BoxDecoration(
        color: entry.isCorrected
            ? VerbaColors.textBlue.withValues(alpha: 0.06)
            : Colors.transparent,
        borderRadius: BorderRadius.circular(8),
        border: entry.isCorrected
            ? const Border(
                left: BorderSide(color: VerbaColors.textBlue, width: 2),
              )
            : null,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            entry.original,
            softWrap: true,
            style: TextStyle(
              color: VerbaColors.inkWhite.withValues(alpha: 0.58),
              fontSize: 12,
              fontWeight: FontWeight.w600,
              height: 1.25,
              decoration: TextDecoration.none,
            ),
          ),
          const SizedBox(height: 4),
          RichText(
            text: TextSpan(children: _translationSpans(entry)),
            softWrap: true,
          ),
        ],
      ),
    );
  }
}

List<TextSpan> _translationSpans(SubtitleEntry entry) {
  final oldText = entry.oldTranslation?.trim();
  if (entry.isCorrected && oldText != null && oldText.isNotEmpty) {
    return [
      TextSpan(
        text: oldText,
        style: _translationStyle(
          color: VerbaColors.mutedGray.withValues(alpha: 0.72),
          decoration: TextDecoration.lineThrough,
        ),
      ),
      TextSpan(
        text: ' ${entry.translation}',
        style: _translationStyle(
          color: VerbaColors.accentYellow,
          weight: FontWeight.w900,
        ),
      ),
    ];
  }

  return [
    TextSpan(
      text: entry.translation,
      style: _translationStyle(color: VerbaColors.inkWhite),
    ),
  ];
}

TextStyle _translationStyle({
  required Color color,
  FontWeight weight = FontWeight.w800,
  TextDecoration decoration = TextDecoration.none,
}) {
  return TextStyle(
    color: color,
    fontSize: 15,
    fontWeight: weight,
    height: 1.25,
    decoration: decoration,
    decorationColor: color.withValues(alpha: 0.82),
    decorationThickness: 1.3,
  );
}
