import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../providers/session_provider.dart';
import '../theme/verba_theme.dart';
import 'subtitle_item.dart';

/// Smooth-scrolling bilingual subtitle list.
///
/// Auto-follows new content when the user is near the bottom. When the user
/// scrolls up to read history, auto-follow pauses and a subtle indicator
/// appears. Tapping the indicator or scrolling back to the bottom resumes.
class SubtitleList extends ConsumerStatefulWidget {
  const SubtitleList({super.key});

  @override
  ConsumerState<SubtitleList> createState() => _SubtitleListState();
}

class _SubtitleListState extends ConsumerState<SubtitleList> {
  final ScrollController _scrollController = ScrollController();
  bool _following = true;

  @override
  void dispose() {
    _scrollController.dispose();
    super.dispose();
  }

  bool get _isNearBottom {
    if (!_scrollController.hasClients) return true;
    final pos = _scrollController.position;
    return pos.pixels >= pos.maxScrollExtent - 60;
  }

  void _scrollToBottom({bool jump = false}) {
    if (!_scrollController.hasClients) return;
    final target = _scrollController.position.maxScrollExtent;
    if (jump) {
      _scrollController.jumpTo(target);
    } else {
      _scrollController.animateTo(
        target,
        duration: const Duration(milliseconds: 250),
        curve: Curves.easeOut,
      );
    }
  }

  void _resumeFollowing() {
    setState(() => _following = true);
    _scrollToBottom();
  }

  @override
  Widget build(BuildContext context) {
    final subtitles = ref.watch(subtitleListProvider);

    // Auto-scroll only when following and new subtitles arrive.
    if (_following && subtitles.isNotEmpty) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (_following) _scrollToBottom();
      });
    }

    if (subtitles.isEmpty) {
      return const Center(
        child: Text(
          '等待语音输入...',
          style: TextStyle(color: VerbaColors.mutedGray, fontSize: 16),
        ),
      );
    }

    return NotificationListener<ScrollNotification>(
      onNotification: (notification) {
        if (notification is ScrollStartNotification &&
            notification.dragDetails != null) {
          // User-initiated scroll — pause follow if scrolling up.
          if (_following && !_isNearBottom) {
            setState(() => _following = false);
          }
        }
        return false;
      },
      child: Stack(
        children: [
          ListView.builder(
            controller: _scrollController,
            padding: const EdgeInsets.fromLTRB(12, 8, 12, 40),
            itemCount: subtitles.length,
            itemBuilder: (context, index) {
              return SubtitleItem(
                entry: subtitles[index],
                isNewest: index == subtitles.length - 1,
              );
            },
          ),

          // "New content below" indicator — subtle pill at the bottom center
          if (!_following)
            Positioned(
              bottom: 10,
              left: 0,
              right: 0,
              child: Center(
                child: GestureDetector(
                  onTap: _resumeFollowing,
                  child: Container(
                    padding: const EdgeInsets.symmetric(
                      horizontal: 16,
                      vertical: 7,
                    ),
                    decoration: BoxDecoration(
                      color: VerbaColors.brandBlue.withValues(alpha: 0.85),
                      borderRadius: BorderRadius.circular(20),
                      boxShadow: [
                        BoxShadow(
                          color: VerbaColors.brandBlue.withValues(alpha: 0.3),
                          blurRadius: 8,
                        ),
                      ],
                    ),
                    child: Row(
                      mainAxisSize: MainAxisSize.min,
                      children: const [
                        Icon(Icons.arrow_downward,
                            size: 14, color: Colors.white),
                        SizedBox(width: 6),
                        Text(
                          '追随最新',
                          style: TextStyle(
                            color: Colors.white,
                            fontSize: 12,
                            fontWeight: FontWeight.w700,
                            decoration: TextDecoration.none,
                          ),
                        ),
                      ],
                    ),
                  ),
                ),
              ),
            ),
        ],
      ),
    );
  }
}
