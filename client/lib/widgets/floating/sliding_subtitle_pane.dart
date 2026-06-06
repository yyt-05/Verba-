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
                      ),
                    ),
                    const Spacer(),
                    Text(
                      '${recent.length} 条',
                      style: const TextStyle(
                        color: VerbaColors.mutedGray,
                        fontSize: 12,
                        fontWeight: FontWeight.w700,
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
      margin: const EdgeInsets.only(bottom: 10),
      padding: const EdgeInsets.fromLTRB(10, 6, 8, 7),
      decoration: BoxDecoration(
        color: entry.isCorrected
            ? VerbaColors.accentYellow.withValues(alpha: 0.06)
            : Colors.transparent,
        borderRadius: BorderRadius.circular(8),
        border: entry.isCorrected
            ? const Border(
                left: BorderSide(color: VerbaColors.accentYellow, width: 3),
              )
            : null,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            entry.original,
            maxLines: 2,
            overflow: TextOverflow.ellipsis,
            style: TextStyle(
              color: VerbaColors.inkWhite.withValues(alpha: 0.58),
              fontSize: 12,
              fontWeight: FontWeight.w600,
            ),
          ),
          const SizedBox(height: 2),
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Expanded(
                child: Text(
                  entry.translation,
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                  style: const TextStyle(
                    color: VerbaColors.inkWhite,
                    fontSize: 15,
                    fontWeight: FontWeight.w800,
                  ),
                ),
              ),
              if (entry.isCorrected) const _CorrectionBadge(),
            ],
          ),
          if (entry.isCorrected && entry.oldTranslation != null)
            Text(
              '原译：${entry.oldTranslation}',
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: TextStyle(
                color: VerbaColors.mutedGray.withValues(alpha: 0.66),
                fontSize: 11,
                fontWeight: FontWeight.w700,
              ),
            ),
        ],
      ),
    );
  }
}

class _CorrectionBadge extends StatelessWidget {
  const _CorrectionBadge();

  @override
  Widget build(BuildContext context) {
    return Container(
      margin: const EdgeInsets.only(left: 8, top: 2),
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
      decoration: BoxDecoration(
        color: VerbaColors.accentYellow.withValues(alpha: 0.14),
        borderRadius: BorderRadius.circular(8),
      ),
      child: const Text(
        '修正',
        style: TextStyle(
          color: VerbaColors.accentYellow,
          fontSize: 10,
          fontWeight: FontWeight.w900,
        ),
      ),
    );
  }
}
