import 'package:flutter/material.dart';
import '../../models/subtitle_entry.dart';
import '../../providers/session_provider.dart';
import '../../theme/verba_theme.dart';
import 'floating_icon.dart';
import 'glass_surface.dart';

class DataConsolePanel extends StatelessWidget {
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
  Widget build(BuildContext context) {
    final updates = subtitles.where((e) => e.isCorrected).length;

    return GlassSurface(
      radius: 16,
      opacity: 0.86,
      padding: EdgeInsets.zero,
      borderColor: VerbaColors.inkWhite,
      child: SizedBox(
        width: 430,
        height: 300,
        child: Column(
          children: [
            Padding(
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
            ),
            Divider(color: Colors.white.withValues(alpha: 0.08), height: 1),
            Padding(
              padding: const EdgeInsets.all(14),
              child: Column(
                children: [
                  Row(
                    children: [
                      _MetricCard(label: '状态', value: _stateLabel(state)),
                      const SizedBox(width: 8),
                      _MetricCard(label: '字幕', value: '${subtitles.length}'),
                      const SizedBox(width: 8),
                      _MetricCard(label: '更新', value: '$updates'),
                    ],
                  ),
                  const SizedBox(height: 14),
                  Row(
                    children: [
                      const FloatingIcon(name: 'verba-audio-source', size: 30),
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
                  Align(
                    alignment: Alignment.centerLeft,
                    child: Text(
                      '字幕会按上下文持续更新；主窗口以整段显示为主。',
                      style: TextStyle(
                        color: VerbaColors.mutedGray.withValues(alpha: 0.9),
                        fontSize: 12,
                        fontWeight: FontWeight.w700,
                        decoration: TextDecoration.none,
                      ),
                    ),
                  ),
                ],
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
