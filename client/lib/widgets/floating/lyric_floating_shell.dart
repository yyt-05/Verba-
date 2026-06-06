import 'dart:async';

import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:window_manager/window_manager.dart';
import '../../models/subtitle_entry.dart';
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
  bool _correctionPreview = false;
  double _fontScale = 1.0;
  int _lastCorrectionKey = 0;
  Timer? _correctionTimer;

  @override
  void dispose() {
    _correctionTimer?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final state = ref.watch(sessionProvider);
    final subtitles = ref.watch(subtitleListProvider);
    final active = _isActive(state);
    _syncCorrectionPreview(subtitles);

    return ColoredBox(
      color: Colors.transparent,
      child: Center(
        child: active
            ? MouseRegion(
                onEnter: (_) => setState(() => _hovering = true),
                onExit: (_) => setState(() => _hovering = false),
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    if (_panel == FloatingPanel.subtitles)
                      SlidingSubtitlePane(
                        subtitles: subtitles,
                        onCollapse: () =>
                            setState(() => _panel = FloatingPanel.lyric),
                      )
                    else if (_panel == FloatingPanel.console)
                      DataConsolePanel(
                        state: state,
                        subtitles: subtitles,
                        onCollapse: () =>
                            setState(() => _panel = FloatingPanel.lyric),
                        onStop: _stopListening,
                        onClear: () =>
                            ref.read(subtitleListProvider.notifier).clear(),
                      )
                    else
                      MouseRegion(
                        cursor: SystemMouseCursors.move,
                        child: GestureDetector(
                          onPanStart: (_) => windowManager.startDragging(),
                          child: Listener(
                            onPointerSignal: (event) {
                              if (event is PointerScrollEvent) {
                                _changeFontScale(
                                  event.scrollDelta.dy < 0 ? 0.06 : -0.06,
                                );
                              }
                            },
                            child: LyricSubtitleCard(
                              subtitles: subtitles,
                              correctionPreview: _correctionPreview,
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
                      opacity: _hovering || _panel != FloatingPanel.lyric
                          ? 1
                          : 0,
                      child: Padding(
                        padding: const EdgeInsets.only(top: 10),
                        child: IgnorePointer(
                          ignoring: !_hovering && _panel == FloatingPanel.lyric,
                          child: FloatingControlBar(
                            subtitleCount: subtitles.length,
                            onOpenSubtitles: () => setState(
                              () => _panel = FloatingPanel.subtitles,
                            ),
                            onOpenConsole: () =>
                                setState(() => _panel = FloatingPanel.console),
                            onFontDown: () => _changeFontScale(-0.1),
                            onFontUp: () => _changeFontScale(0.1),
                            onStop: _stopListening,
                          ),
                        ),
                      ),
                    ),
                  ],
                ),
              )
            : MiniFloatingPill(
                onStart: () =>
                    ref.read(sessionProvider.notifier).startListening(),
              ),
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
      _correctionPreview = false;
    });
    ref.read(sessionProvider.notifier).stopListening();
  }

  void _changeFontScale(double delta) {
    setState(() {
      _fontScale = (_fontScale + delta).clamp(0.7, 1.65).toDouble();
    });
  }

  void _syncCorrectionPreview(List<SubtitleEntry> subtitles) {
    var key = 0;
    for (final entry in subtitles) {
      if (entry.isCorrected) {
        key = entry.segmentId * 1000 + entry.revision;
      }
    }
    if (key == 0 || key == _lastCorrectionKey) return;
    _lastCorrectionKey = key;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      setState(() {
        _panel = FloatingPanel.lyric;
        _correctionPreview = true;
      });
      _correctionTimer?.cancel();
      _correctionTimer = Timer(const Duration(milliseconds: 2600), () {
        if (mounted) setState(() => _correctionPreview = false);
      });
    });
  }
}
