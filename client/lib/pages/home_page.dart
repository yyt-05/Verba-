import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../providers/session_provider.dart';
import '../widgets/audio_meter.dart';
import '../widgets/subtitle_list.dart';

class HomePage extends ConsumerStatefulWidget {
  const HomePage({super.key});

  @override
  ConsumerState<HomePage> createState() => _HomePageState();
}

class _HomePageState extends ConsumerState<HomePage> {
  @override
  Widget build(BuildContext context) {
    final sessionState = ref.watch(sessionProvider);
    final subtitleCount = ref.watch(subtitleListProvider).length;

    return Container(
      color: const Color(0xDD000000),
      child: Column(
        children: [
          const SizedBox(height: 32),
          const Text('Verba',
            style: TextStyle(color: Colors.white, fontSize: 18, fontWeight: FontWeight.bold)),
          const Text('AI 实时双语字幕助手',
            style: TextStyle(color: Colors.white38, fontSize: 12)),

          const SizedBox(height: 12),

          // Subtitle list (Phase 0: shows placeholder subtitles from mock uploads)
          Expanded(
            child: sessionState == SessionState.listening
                ? const SubtitleList()
                : Center(
                    child: Text(
                      'Phase 0 - WASAPI 音频捕获测试\n点右侧蓝色按钮连接后端',
                      textAlign: TextAlign.center,
                      style: TextStyle(color: Colors.white.withValues(alpha: 0.4), fontSize: 13),
                    ),
                  ),
          ),

          // Status bar
          if (sessionState == SessionState.listening)
            Padding(
              padding: const EdgeInsets.only(bottom: 4),
              child: Text(
                '已收到 $subtitleCount 条字幕',
                style: const TextStyle(color: Colors.white24, fontSize: 10),
              ),
            ),

          // Bottom buttons
          Padding(
            padding: const EdgeInsets.only(bottom: 48),
            child: Row(
              mainAxisAlignment: MainAxisAlignment.spaceEvenly,
              children: [
                const AudioMeter(),
                _ServerButton(state: sessionState),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _ServerButton extends ConsumerWidget {
  final SessionState state;
  const _ServerButton({required this.state});

  bool get _isActive =>
      state == SessionState.listening || state == SessionState.reconnecting;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final notifier = ref.read(sessionProvider.notifier);

    return GestureDetector(
      onTap: () {
        if (_isActive) {
          notifier.stopListening();
        } else {
          notifier.startListening();
        }
      },
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          AnimatedContainer(
            duration: const Duration(milliseconds: 300),
            width: 56, height: 56,
            decoration: BoxDecoration(
              shape: BoxShape.circle,
              color: _isActive ? const Color(0xFFE53935) : const Color(0xFF1A73E8),
              boxShadow: [
                BoxShadow(
                  color: (_isActive ? const Color(0xFFE53935) : const Color(0xFF1A73E8))
                      .withValues(alpha: 0.4),
                  blurRadius: 12, spreadRadius: 2,
                ),
              ],
            ),
            child: Icon(_isActive ? Icons.stop : Icons.mic, color: Colors.white, size: 28),
          ),
          const SizedBox(height: 6),
          Text(
            _isActive ? '停止服务' : '服务连接',
            style: TextStyle(
              color: _isActive ? Colors.redAccent : Colors.white54, fontSize: 11),
          ),
        ],
      ),
    );
  }
}
