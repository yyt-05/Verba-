import 'package:flutter/material.dart';
import '../../theme/verba_theme.dart';
import 'floating_icon.dart';
import 'glass_surface.dart';

class FloatingControlBar extends StatelessWidget {
  final int subtitleCount;
  final bool ttsEnabled;
  final VoidCallback onOpenSubtitles;
  final VoidCallback onOpenConsole;
  final VoidCallback onFontDown;
  final VoidCallback onFontUp;
  final VoidCallback onToggleTts;
  final VoidCallback onStop;

  const FloatingControlBar({
    super.key,
    required this.subtitleCount,
    required this.ttsEnabled,
    required this.onOpenSubtitles,
    required this.onOpenConsole,
    required this.onFontDown,
    required this.onFontUp,
    required this.onToggleTts,
    required this.onStop,
  });

  @override
  Widget build(BuildContext context) {
    return GlassSurface(
      radius: 22,
      opacity: 0.18,
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 7),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          const FloatingIcon(name: 'verba-audio-source', size: 26),
          const SizedBox(width: 8),
          Text(
            '$subtitleCount 条',
            style: const TextStyle(
              color: VerbaColors.inkWhite,
              fontSize: 12,
              fontWeight: FontWeight.w800,
              decoration: TextDecoration.none,
            ),
          ),
          const SizedBox(width: 10),
          _IconButton(
            name: 'verba-tts',
            active: ttsEnabled,
            tooltip: ttsEnabled ? '关闭中文朗读' : '开启中文朗读',
            onTap: onToggleTts,
          ),
          _IconButton(name: 'verba-font-decrease', onTap: onFontDown),
          _IconButton(name: 'verba-font-increase', onTap: onFontUp),
          _IconButton(name: 'verba-subtitle-pane', onTap: onOpenSubtitles),
          _IconButton(name: 'verba-dashboard', onTap: onOpenConsole),
          _IconButton(name: 'verba-stop', onTap: onStop),
        ],
      ),
    );
  }
}

class _IconButton extends StatelessWidget {
  final String name;
  final bool active;
  final String? tooltip;
  final VoidCallback onTap;

  const _IconButton({
    required this.name,
    required this.onTap,
    this.active = false,
    this.tooltip,
  });

  @override
  Widget build(BuildContext context) {
    final icon = AnimatedContainer(
      duration: const Duration(milliseconds: 160),
      padding: const EdgeInsets.all(2),
      decoration: BoxDecoration(
        shape: BoxShape.circle,
        color: active
            ? VerbaColors.successGreen.withValues(alpha: 0.22)
            : Colors.transparent,
        boxShadow: active
            ? [
                BoxShadow(
                  color: VerbaColors.successGreen.withValues(alpha: 0.34),
                  blurRadius: 12,
                ),
              ]
            : null,
      ),
      child: FloatingIcon(name: name, size: 28),
    );

    return Padding(
      padding: const EdgeInsets.only(left: 4),
      child: GestureDetector(
        onTap: onTap,
        child: MouseRegion(
          cursor: SystemMouseCursors.click,
          child: tooltip == null
              ? icon
              : Tooltip(message: tooltip!, child: icon),
        ),
      ),
    );
  }
}
