import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../providers/session_provider.dart';
import 'subtitle_list.dart';

/// The expanded subtitle area shown when session is active.
class ExpandedSubtitleArea extends ConsumerWidget {
  final SessionState state;

  const ExpandedSubtitleArea({super.key, required this.state});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final isActive = state == SessionState.listening ||
        state == SessionState.silent ||
        state == SessionState.reconnecting ||
        state == SessionState.stopped;

    if (!isActive) return const SizedBox.shrink();

    return Positioned(
      bottom: 100,
      left: 0,
      right: 0,
      top: 40,
      child: Container(
        margin: const EdgeInsets.symmetric(horizontal: 16),
        decoration: BoxDecoration(
          color: Colors.black.withValues(alpha: 0.75),
          borderRadius: BorderRadius.circular(12),
        ),
        child: const ClipRRect(
          borderRadius: BorderRadius.all(Radius.circular(12)),
          child: SubtitleList(),
        ),
      ),
    );
  }
}

/// The main floating action button that toggles listening.
class FloatingControlButton extends ConsumerWidget {
  final SessionState state;

  const FloatingControlButton({super.key, required this.state});

  bool get isActive => state == SessionState.listening ||
      state == SessionState.silent ||
      state == SessionState.reconnecting;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final notifier = ref.read(sessionProvider.notifier);

    return GestureDetector(
      onTap: () {
        if (isActive) {
          notifier.stopListening();
        } else {
          notifier.startListening();
        }
      },
      child: AnimatedContainer(
        duration: const Duration(milliseconds: 300),
        width: isActive ? 56 : 64,
        height: isActive ? 56 : 64,
        decoration: BoxDecoration(
          shape: BoxShape.circle,
          color: isActive
              ? const Color(0xFFE53935) // Red when active (stop action)
              : const Color(0xFF1A73E8), // Blue when idle (start action)
          boxShadow: [
            BoxShadow(
              color: (isActive ? const Color(0xFFE53935) : const Color(0xFF1A73E8))
                  .withValues(alpha: 0.4),
              blurRadius: 12,
              spreadRadius: 2,
            ),
          ],
        ),
        child: Icon(
          isActive ? Icons.stop : Icons.mic,
          color: Colors.white,
          size: 28,
        ),
      ),
    );
  }
}
