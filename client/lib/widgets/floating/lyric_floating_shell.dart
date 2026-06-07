import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:window_manager/window_manager.dart';
import '../../providers/session_provider.dart';
import 'data_console_panel.dart';
import 'floating_control_bar.dart';
import 'lyric_subtitle_card.dart';
import 'mini_floating_pill.dart';
import 'sliding_subtitle_pane.dart';

enum FloatingPanel { lyric, subtitles, console }

class LyricFloatingShell extends ConsumerStatefulWidget {
  const LyricFloatingShell({super.key});

  @override
  ConsumerState<LyricFloatingShell> createState() => _LyricFloatingShellState();
}

class _LyricFloatingShellState extends ConsumerState<LyricFloatingShell> {
  FloatingPanel _panel = FloatingPanel.lyric;
  bool _hovering = false;
  bool _ttsEnabled = false;
  double _fontScale = 1.0;

  @override
  Widget build(BuildContext context) {
    final state = ref.watch(sessionProvider);
    final subtitles = ref.watch(subtitleListProvider);
    final active = _isActive(state);

    return ColoredBox(
      color: Colors.transparent,
      child: LayoutBuilder(
        builder: (context, constraints) {
          final maxHeight = constraints.maxHeight.isFinite
              ? constraints.maxHeight
              : MediaQuery.of(context).size.height;

          return Center(
            child: active
                ? MouseRegion(
                    onEnter: (_) => setState(() => _hovering = true),
                    onExit: (_) => setState(() => _hovering = false),
                    child: ConstrainedBox(
                      constraints: BoxConstraints(maxHeight: maxHeight),
                      child: SingleChildScrollView(
                        padding: const EdgeInsets.symmetric(vertical: 8),
                        child: Column(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            if (_panel == FloatingPanel.subtitles)
                              SlidingSubtitlePane(
                                subtitles: subtitles,
                                onCollapse: () => setState(
                                  () => _panel = FloatingPanel.lyric,
                                ),
                              )
                            else if (_panel == FloatingPanel.console)
                              DataConsolePanel(
                                state: state,
                                subtitles: subtitles,
                                onCollapse: () => setState(
                                  () => _panel = FloatingPanel.lyric,
                                ),
                                onStop: _stopListening,
                                onClear: () => ref
                                    .read(subtitleListProvider.notifier)
                                    .clear(),
                              )
                            else
                              MouseRegion(
                                cursor: SystemMouseCursors.move,
                                child: GestureDetector(
                                  onPanStart: (_) =>
                                      windowManager.startDragging(),
                                  child: Listener(
                                    onPointerSignal: (event) {
                                      if (event is PointerScrollEvent) {
                                        _changeFontScale(
                                          event.scrollDelta.dy < 0
                                              ? 0.06
                                              : -0.06,
                                        );
                                      }
                                    },
                                    child: LyricSubtitleCard(
                                      subtitles: subtitles,
                                      correctionPreview: true,
                                      fontScale: _fontScale,
                                      onTap: () => setState(
                                        () => _panel = FloatingPanel.subtitles,
                                      ),
                                    ),
                                  ),
                                ),
                              ),
                            AnimatedOpacity(
                              duration: const Duration(milliseconds: 160),
                              opacity:
                                  _hovering || _panel != FloatingPanel.lyric
                                  ? 1
                                  : 0,
                              child: Padding(
                                padding: const EdgeInsets.only(top: 10),
                                child: IgnorePointer(
                                  ignoring:
                                      !_hovering &&
                                      _panel == FloatingPanel.lyric,
                                  child: FloatingControlBar(
                                    subtitleCount: subtitles.length,
                                    ttsEnabled: _ttsEnabled,
                                    onOpenSubtitles: () => setState(
                                      () => _panel = FloatingPanel.subtitles,
                                    ),
                                    onOpenConsole: () => setState(
                                      () => _panel = FloatingPanel.console,
                                    ),
                                    onFontDown: () => _changeFontScale(-0.1),
                                    onFontUp: () => _changeFontScale(0.1),
                                    onToggleTts: _toggleTts,
                                    onStop: _stopListening,
                                  ),
                                ),
                              ),
                            ),
                          ],
                        ),
                      ),
                    ),
                  )
                : MiniFloatingPill(
                    onStart: () =>
                        ref.read(sessionProvider.notifier).startListening(),
                  ),
          );
        },
      ),
    );
  }

  bool _isActive(SessionState state) {
    return state == SessionState.connecting ||
        state == SessionState.listening ||
        state == SessionState.silent ||
        state == SessionState.reconnecting;
  }

  void _stopListening() {
    setState(() {
      _panel = FloatingPanel.lyric;
    });
    ref.read(sessionProvider.notifier).stopListening();
  }

  void _changeFontScale(double delta) {
    setState(() {
      _fontScale = (_fontScale + delta).clamp(0.7, 1.65).toDouble();
    });
  }

  Future<void> _toggleTts() async {
    final next = !_ttsEnabled;
    final ok = await ref.read(sessionProvider.notifier).setTtsEnabled(next);
    if (!mounted) return;
    setState(() {
      _ttsEnabled = ok ? next : false;
    });
  }
}
