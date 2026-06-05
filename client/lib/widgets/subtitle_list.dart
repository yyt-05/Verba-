import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../providers/session_provider.dart';
import '../theme/verba_theme.dart';
import 'subtitle_item.dart';

/// Scrollable bilingual subtitle list with auto-scroll to bottom.
class SubtitleList extends ConsumerStatefulWidget {
  const SubtitleList({super.key});

  @override
  ConsumerState<SubtitleList> createState() => _SubtitleListState();
}

class _SubtitleListState extends ConsumerState<SubtitleList> {
  final ScrollController _scrollController = ScrollController();
  bool _autoScroll = true;

  @override
  void dispose() {
    _scrollController.dispose();
    super.dispose();
  }

  void _scrollToBottom() {
    if (_autoScroll && _scrollController.hasClients) {
      _scrollController.animateTo(
        _scrollController.position.maxScrollExtent,
        duration: const Duration(milliseconds: 200),
        curve: Curves.easeOut,
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final subtitles = ref.watch(subtitleListProvider);

    // Auto-scroll when new subtitles arrive
    WidgetsBinding.instance.addPostFrameCallback((_) => _scrollToBottom());

    if (subtitles.isEmpty) {
      return const Center(
        child: Text(
          '点击下方按钮开始监听',
          style: TextStyle(color: VerbaColors.mutedGray, fontSize: 16),
        ),
      );
    }

    return NotificationListener<ScrollNotification>(
      onNotification: (notification) {
        if (notification is ScrollUpdateNotification && notification.dragDetails != null) {
          // User is manually scrolling — pause auto-scroll
          _autoScroll = false;
        }
        return false;
      },
      child: Stack(
        children: [
          ListView.builder(
            controller: _scrollController,
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
            itemCount: subtitles.length,
            itemBuilder: (context, index) {
              return SubtitleItem(
                entry: subtitles[index],
                isNewest: index == subtitles.length - 1,
              );
            },
          ),
          // "Scroll to bottom" button when user has scrolled up
          if (!_autoScroll)
            Positioned(
              bottom: 8,
              right: 8,
              child: FloatingActionButton.small(
                onPressed: () {
                  _autoScroll = true;
                  _scrollToBottom();
                },
                backgroundColor: VerbaColors.brandBlue,
                child: const Icon(Icons.arrow_downward, color: VerbaColors.inkWhite),
              ),
            ),
        ],
      ),
    );
  }
}
