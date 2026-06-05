import 'dart:async';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../providers/session_provider.dart';
import '../theme/verba_theme.dart';

/// WASAPI audio capture toggle + volume meter.
/// Uses the shared WasapiCapture instance from Riverpod provider.
class AudioMeter extends ConsumerStatefulWidget {
  const AudioMeter({super.key});

  @override
  ConsumerState<AudioMeter> createState() => _AudioMeterState();
}

class _AudioMeterState extends ConsumerState<AudioMeter> {
  Timer? _pollTimer;
  double _level = 0.0;
  bool _capturing = false;
  int? _errorCode;

  @override
  void dispose() {
    _pollTimer?.cancel();
    if (_capturing) ref.read(wasapiProvider).stop();
    super.dispose();
  }

  void _toggle() {
    final wasapi = ref.read(wasapiProvider);
    if (_capturing) {
      _pollTimer?.cancel();
      wasapi.stop();
      setState(() {
        _capturing = false;
        _level = 0.0;
        _errorCode = null;
      });
    } else {
      final result = wasapi.start();
      if (result == 0) {
        setState(() {
          _capturing = true;
          _errorCode = null;
        });
        _pollTimer = Timer.periodic(const Duration(milliseconds: 50), (_) {
          setState(() {
            _level = wasapi.level;
          });
        });
      } else {
        setState(() {
          _errorCode = result;
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final wasapi = ref.watch(wasapiProvider);

    return GestureDetector(
      onTap: _toggle,
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (_errorCode != null)
            Container(
              margin: const EdgeInsets.only(bottom: 8),
              padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
              decoration: BoxDecoration(
                color: VerbaColors.dangerRed,
                borderRadius: BorderRadius.circular(VerbaTheme.panelRadius),
              ),
              child: Text(
                'WASAPI 错误 (code=$_errorCode)',
                style: const TextStyle(color: Colors.white, fontSize: 11),
              ),
            ),

          // Volume bar
          if (_capturing)
            Container(
              width: 48, height: 96,
              margin: const EdgeInsets.only(bottom: 8),
              decoration: BoxDecoration(
                color: Colors.black38,
                borderRadius: BorderRadius.circular(VerbaTheme.panelRadius),
                border: Border.all(color: VerbaColors.softBlue.withValues(alpha: 0.18)),
              ),
              child: Align(
                alignment: Alignment.bottomCenter,
                child: AnimatedContainer(
                  duration: const Duration(milliseconds: 80),
                  curve: Curves.easeOut,
                  width: 44,
                  height: 92 * _level.clamp(0.0, 1.0),
                  decoration: const BoxDecoration(
                    gradient: LinearGradient(
                      begin: Alignment.bottomCenter,
                      end: Alignment.topCenter,
                      colors: [
                        VerbaColors.successGreen,
                        VerbaColors.accentYellow,
                        VerbaColors.dangerRed,
                      ],
                    ),
                    borderRadius: BorderRadius.all(Radius.circular(6)),
                  ),
                ),
              ),
            ),

          // Button
          AnimatedContainer(
            duration: const Duration(milliseconds: 300),
            width: VerbaTheme.controlSize, height: VerbaTheme.controlSize,
            decoration: BoxDecoration(
              shape: BoxShape.circle,
              color: _capturing ? VerbaColors.dangerRed : VerbaColors.successGreen,
              boxShadow: [
                BoxShadow(
                  color: (_capturing ? VerbaColors.dangerRed : VerbaColors.successGreen)
                      .withValues(alpha: 0.5),
                  blurRadius: VerbaTheme.buttonShadowBlur,
                  spreadRadius: 1,
                ),
              ],
            ),
            child: Icon(
              _capturing ? Icons.stop : Icons.headset_mic,
              color: Colors.white, size: 28,
            ),
          ),
          const SizedBox(height: 6),
          Text(
            _errorCode != null
                ? '启动失败'
                : _capturing
                    ? '正在捕获系统音频'
                    : '点击测试 WASAPI',
            style: TextStyle(
              color: _capturing ? VerbaColors.successGreen : VerbaColors.mutedGray,
              fontSize: 11,
              fontWeight: FontWeight.w700,
            ),
          ),
          if (_capturing)
            Text(
              '电平: ${(_level * 100).toStringAsFixed(0)}%',
              style: const TextStyle(color: VerbaColors.mutedGray, fontSize: 10),
            ),
          if (_capturing)
            Padding(
              padding: const EdgeInsets.only(top: 4),
              child: Container(
                padding: const EdgeInsets.all(6),
                decoration: BoxDecoration(
                  color: Colors.white.withValues(alpha: 0.08),
                  borderRadius: BorderRadius.circular(VerbaTheme.panelRadius),
                ),
                child: Text(wasapi.diag,
                  style: const TextStyle(color: Colors.white38, fontSize: 9, fontFamily: 'monospace')),
              ),
            ),
        ],
      ),
    );
  }
}
