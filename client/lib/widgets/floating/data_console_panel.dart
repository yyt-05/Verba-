import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../models/subtitle_entry.dart';
import '../../providers/session_provider.dart';
import '../../theme/verba_theme.dart';
import 'floating_icon.dart';
import 'glass_surface.dart';

class DataConsolePanel extends ConsumerStatefulWidget {
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
  ConsumerState<DataConsolePanel> createState() => _DataConsolePanelState();
}

class _DataConsolePanelState extends ConsumerState<DataConsolePanel> {
  Timer? _timer;
  double _level = 0;
  int _availableBytes = 0;
  bool _capturing = false;
  String _diag = '';

  @override
  void initState() {
    super.initState();
    _readAudioState();
    _timer = Timer.periodic(const Duration(milliseconds: 250), (_) {
      if (mounted) _readAudioState();
    });
  }

  @override
  void dispose() {
    _timer?.cancel();
    super.dispose();
  }

  void _readAudioState() {
    final wasapi = ref.read(wasapiProvider);
    setState(() {
      _capturing = wasapi.isCapturing;
      _availableBytes = wasapi.availableBytes;
      _level = wasapi.level.clamp(0.0, 1.0).toDouble();
      _diag = wasapi.diag;
    });
  }

  @override
  Widget build(BuildContext context) {
    final corrections = widget.subtitles.where((e) => e.isCorrected).length;
    final receiving = _capturing && _availableBytes > 0;

    return GlassSurface(
      radius: 16,
      opacity: 0.86,
      padding: EdgeInsets.zero,
      borderColor: VerbaColors.inkWhite,
      child: SizedBox(
        width: 460,
        height: 340,
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
                    ),
                  ),
                  const Spacer(),
                  GestureDetector(
                    onTap: widget.onCollapse,
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
                      _MetricCard(
                        label: '状态',
                        value: _stateLabel(widget.state),
                      ),
                      const SizedBox(width: 8),
                      _MetricCard(
                        label: '字幕',
                        value: '${widget.subtitles.length}',
                      ),
                      const SizedBox(width: 8),
                      _MetricCard(label: '修正', value: '$corrections'),
                    ],
                  ),
                  const SizedBox(height: 12),
                  _AudioStatusCard(
                    capturing: _capturing,
                    receiving: receiving,
                    level: _level,
                    availableBytes: _availableBytes,
                    diag: _diag,
                  ),
                  const SizedBox(height: 12),
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
                          ),
                        ),
                      ),
                      _ActionButton(label: '清空', onTap: widget.onClear),
                      const SizedBox(width: 8),
                      _ActionButton(
                        label: '停止',
                        danger: true,
                        onTap: widget.onStop,
                      ),
                    ],
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
      SessionState.listening => '监听',
      SessionState.reconnecting => '重连',
      SessionState.stopped => '停止',
      SessionState.apiError => '错误',
      SessionState.audioSourceUnavailable => '无音频',
      _ => '待机',
    };
  }
}

class _AudioStatusCard extends StatelessWidget {
  final bool capturing;
  final bool receiving;
  final double level;
  final int availableBytes;
  final String diag;

  const _AudioStatusCard({
    required this.capturing,
    required this.receiving,
    required this.level,
    required this.availableBytes,
    required this.diag,
  });

  @override
  Widget build(BuildContext context) {
    final status = !capturing
        ? '未捕获'
        : receiving
        ? '收到系统音频'
        : '捕获中，无可读音频';
    final color = receiving
        ? VerbaColors.successGreen
        : capturing
        ? VerbaColors.accentYellow
        : VerbaColors.mutedGray;

    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: Colors.white.withValues(alpha: 0.08),
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: color.withValues(alpha: 0.3)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Container(
                width: 9,
                height: 9,
                decoration: BoxDecoration(color: color, shape: BoxShape.circle),
              ),
              const SizedBox(width: 8),
              Text(
                status,
                style: TextStyle(
                  color: color,
                  fontSize: 13,
                  fontWeight: FontWeight.w900,
                ),
              ),
              const Spacer(),
              Text(
                '音量 ${(level * 100).toStringAsFixed(0)}%',
                style: const TextStyle(
                  color: VerbaColors.inkWhite,
                  fontSize: 12,
                  fontWeight: FontWeight.w800,
                ),
              ),
            ],
          ),
          const SizedBox(height: 8),
          ClipRRect(
            borderRadius: BorderRadius.circular(4),
            child: LinearProgressIndicator(
              minHeight: 7,
              value: level.clamp(0.0, 1.0),
              backgroundColor: Colors.white.withValues(alpha: 0.08),
              color: color,
            ),
          ),
          const SizedBox(height: 8),
          Text(
            'buffer: $availableBytes bytes  |  $diag',
            maxLines: 2,
            overflow: TextOverflow.ellipsis,
            style: const TextStyle(
              color: VerbaColors.mutedGray,
              fontSize: 11,
              fontWeight: FontWeight.w700,
            ),
          ),
        ],
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
              style: const TextStyle(
                color: VerbaColors.mutedGray,
                fontSize: 11,
                fontWeight: FontWeight.w700,
              ),
            ),
            const SizedBox(height: 4),
            Text(
              value,
              style: const TextStyle(
                color: VerbaColors.inkWhite,
                fontSize: 22,
                fontWeight: FontWeight.w900,
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
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
        decoration: BoxDecoration(
          color: danger
              ? VerbaColors.dangerRed
              : Colors.white.withValues(alpha: 0.12),
          borderRadius: BorderRadius.circular(18),
        ),
        child: Text(
          label,
          style: const TextStyle(
            color: VerbaColors.inkWhite,
            fontSize: 12,
            fontWeight: FontWeight.w900,
          ),
        ),
      ),
    );
  }
}
