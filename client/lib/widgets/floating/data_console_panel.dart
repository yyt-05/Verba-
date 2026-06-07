import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../models/subtitle_entry.dart';
import '../../providers/session_provider.dart';
import '../../theme/verba_theme.dart';
import '../../utils/translation_correction_diff.dart';
import 'floating_icon.dart';
import 'glass_surface.dart';

class DataConsolePanel extends ConsumerWidget {
  final SessionState state;
  final List<SubtitleEntry> subtitles;
  final VoidCallback onCollapse;
  final VoidCallback onStop;
  final VoidCallback onClear;

  const DataConsolePanel({
    super.key,
    required this.state,
    required this.subtitles,
    required this.onCollapse,
    required this.onStop,
    required this.onClear,
  });

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final background = ref.watch(backgroundSummaryProvider);
    final updates = subtitles.where((e) => e.isCorrected).length;

    return GlassSurface(
      radius: 16,
      opacity: 0.88,
      padding: EdgeInsets.zero,
      borderColor: VerbaColors.inkWhite,
      child: SizedBox(
        width: 430,
        height: 400,
        child: Column(
          children: [
            _Header(onCollapse: onCollapse),
            Divider(color: Colors.white.withValues(alpha: 0.08), height: 1),
            Expanded(
              child: SingleChildScrollView(
                padding: const EdgeInsets.all(14),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    // Metrics row
                    Row(
                      children: [
                        _MetricCard(label: '状态', value: _stateLabel(state)),
                        const SizedBox(width: 8),
                        _MetricCard(label: '字幕', value: '${subtitles.length}'),
                        const SizedBox(width: 8),
                        _MetricCard(label: '修正', value: '$updates'),
                      ],
                    ),
                    const SizedBox(height: 12),

                    // Background summary
                    _SectionHeader(title: '背景总结'),
                    const SizedBox(height: 6),
                    _BackgroundCard(summary: background),
                    const SizedBox(height: 12),

                    // Controls
                    Row(
                      children: [
                        const FloatingIcon(
                          name: 'verba-audio-source',
                          size: 30,
                        ),
                        const SizedBox(width: 8),
                        const Expanded(
                          child: Text(
                            'WASAPI 系统音频',
                            style: TextStyle(
                              color: VerbaColors.inkWhite,
                              fontSize: 13,
                              fontWeight: FontWeight.w800,
                              decoration: TextDecoration.none,
                            ),
                          ),
                        ),
                        _ActionButton(label: '清空', onTap: onClear),
                        const SizedBox(width: 8),
                        _ActionButton(label: '停止', danger: true, onTap: onStop),
                      ],
                    ),
                    const SizedBox(height: 14),

                    // Sentence history
                    _SectionHeader(title: '翻译记录 (${subtitles.length})'),
                    const SizedBox(height: 6),
                    if (subtitles.isEmpty)
                      Text(
                        '暂无翻译记录',
                        style: TextStyle(
                          color: VerbaColors.mutedGray.withValues(alpha: 0.7),
                          fontSize: 13,
                          fontWeight: FontWeight.w600,
                          decoration: TextDecoration.none,
                        ),
                      )
                    else
                      ...subtitles.reversed
                          .take(50)
                          .map((e) => _SentenceRow(entry: e)),
                  ],
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }

  String _stateLabel(SessionState state) {
    return switch (state) {
      SessionState.connecting => '连接中',
      SessionState.listening => '监听中',
      SessionState.reconnecting => '重连中',
      SessionState.stopped => '已停止',
      SessionState.apiError => '接口错误',
      SessionState.audioSourceUnavailable => '无音频',
      _ => '待机',
    };
  }
}

class _Header extends StatelessWidget {
  final VoidCallback onCollapse;

  const _Header({required this.onCollapse});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 12, 12, 8),
      child: Row(
        children: [
          const Text(
            'Verba 控制台',
            style: TextStyle(
              color: VerbaColors.inkWhite,
              fontSize: 17,
              fontWeight: FontWeight.w900,
              decoration: TextDecoration.none,
            ),
          ),
          const Spacer(),
          GestureDetector(
            onTap: onCollapse,
            child: const FloatingIcon(name: 'verba-collapse', size: 28),
          ),
        ],
      ),
    );
  }
}

class _SectionHeader extends StatelessWidget {
  final String title;

  const _SectionHeader({required this.title});

  @override
  Widget build(BuildContext context) {
    return Text(
      title,
      style: TextStyle(
        color: VerbaColors.softBlue.withValues(alpha: 0.85),
        fontSize: 12,
        fontWeight: FontWeight.w800,
        decoration: TextDecoration.none,
      ),
    );
  }
}

class _BackgroundCard extends StatelessWidget {
  final String summary;

  const _BackgroundCard({required this.summary});

  @override
  Widget build(BuildContext context) {
    final hasSummary = summary.isNotEmpty;

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: hasSummary
            ? VerbaColors.accentYellow.withValues(alpha: 0.08)
            : Colors.white.withValues(alpha: 0.04),
        borderRadius: BorderRadius.circular(10),
        border: Border.all(
          color: hasSummary
              ? VerbaColors.accentYellow.withValues(alpha: 0.2)
              : Colors.white.withValues(alpha: 0.06),
        ),
      ),
      child: Text(
        hasSummary ? summary : '等待 AI 自动总结...\n\n积累 10 句对话后会自动生成背景总结，帮助翻译更精准。',
        style: TextStyle(
          color: hasSummary
              ? VerbaColors.inkWhite.withValues(alpha: 0.9)
              : VerbaColors.mutedGray.withValues(alpha: 0.7),
          fontSize: 12,
          fontWeight: FontWeight.w600,
          height: 1.5,
          decoration: TextDecoration.none,
        ),
      ),
    );
  }
}

class _SentenceRow extends StatelessWidget {
  final SubtitleEntry entry;

  const _SentenceRow({required this.entry});

  @override
  Widget build(BuildContext context) {
    return Container(
      margin: const EdgeInsets.only(bottom: 6),
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 7),
      decoration: BoxDecoration(
        color: entry.isCorrected
            ? VerbaColors.accentYellow.withValues(alpha: 0.06)
            : Colors.white.withValues(alpha: 0.03),
        borderRadius: BorderRadius.circular(8),
        border: entry.isCorrected
            ? Border.all(
                color: VerbaColors.accentYellow.withValues(alpha: 0.15),
              )
            : null,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Text(
                '#${entry.segmentId}',
                style: TextStyle(
                  color: VerbaColors.mutedGray.withValues(alpha: 0.6),
                  fontSize: 10,
                  fontWeight: FontWeight.w700,
                  decoration: TextDecoration.none,
                ),
              ),
              if (entry.isCorrected) ...[
                const SizedBox(width: 6),
                Icon(
                  Icons.auto_fix_high,
                  size: 11,
                  color: VerbaColors.accentYellow.withValues(alpha: 0.8),
                ),
              ],
            ],
          ),
          const SizedBox(height: 3),
          Text(
            entry.original,
            style: TextStyle(
              color: VerbaColors.inkWhite.withValues(alpha: 0.6),
              fontSize: 12,
              fontWeight: FontWeight.w600,
              height: 1.3,
              decoration: TextDecoration.none,
            ),
          ),
          const SizedBox(height: 2),
          entry.isCorrected &&
                  entry.oldTranslation != null &&
                  entry.oldTranslation!.isNotEmpty
              ? _CorrectionTranslationText(
                  oldText: entry.oldTranslation!,
                  newText: entry.translation,
                )
              : Text(
                  entry.translation,
                  style: TextStyle(
                    color: entry.isCorrected
                        ? VerbaColors.accentYellow.withValues(alpha: 0.9)
                        : VerbaColors.inkWhite.withValues(alpha: 0.85),
                    fontSize: 13,
                    fontWeight: FontWeight.w700,
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

  const _CorrectionTranslationText({
    required this.oldText,
    required this.newText,
  });

  @override
  Widget build(BuildContext context) {
    final baseStyle = TextStyle(
      color: VerbaColors.inkWhite.withValues(alpha: 0.85),
      fontSize: 13,
      fontWeight: FontWeight.w700,
      height: 1.35,
      decoration: TextDecoration.none,
    );

    return RichText(
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
                  style: baseStyle.copyWith(
                    color: VerbaColors.accentYellow.withValues(alpha: 0.9),
                  ),
                );
              }
              return TextSpan(text: part.text);
            }).toList(),
      ),
    );
  }
}

class _MetricCard extends StatelessWidget {
  final String label;
  final String value;

  const _MetricCard({required this.label, required this.value});

  @override
  Widget build(BuildContext context) {
    return Expanded(
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
        decoration: BoxDecoration(
          color: Colors.white.withValues(alpha: 0.08),
          borderRadius: BorderRadius.circular(10),
          border: Border.all(
            color: VerbaColors.softBlue.withValues(alpha: 0.12),
          ),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              label,
              style: TextStyle(
                color: VerbaColors.mutedGray.withValues(alpha: 0.9),
                fontSize: 11,
                fontWeight: FontWeight.w700,
                decoration: TextDecoration.none,
              ),
            ),
            const SizedBox(height: 5),
            Text(
              value,
              style: const TextStyle(
                color: VerbaColors.inkWhite,
                fontSize: 16,
                fontWeight: FontWeight.w900,
                decoration: TextDecoration.none,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _ActionButton extends StatelessWidget {
  final String label;
  final bool danger;
  final VoidCallback onTap;

  const _ActionButton({
    required this.label,
    required this.onTap,
    this.danger = false,
  });

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: onTap,
      child: MouseRegion(
        cursor: SystemMouseCursors.click,
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 7),
          decoration: BoxDecoration(
            color: danger
                ? VerbaColors.dangerRed.withValues(alpha: 0.18)
                : Colors.white.withValues(alpha: 0.08),
            borderRadius: BorderRadius.circular(8),
            border: Border.all(
              color: danger
                  ? VerbaColors.dangerRed.withValues(alpha: 0.35)
                  : Colors.white.withValues(alpha: 0.08),
            ),
          ),
          child: Text(
            label,
            style: TextStyle(
              color: danger ? VerbaColors.dangerRed : VerbaColors.inkWhite,
              fontSize: 12,
              fontWeight: FontWeight.w800,
              decoration: TextDecoration.none,
            ),
          ),
        ),
      ),
    );
  }
}
