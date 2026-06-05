import 'package:flutter/material.dart';
import '../models/subtitle_entry.dart';

/// Single subtitle entry: English original (white) + Chinese translation (highlighted).
///
/// On correction: blue background flash animation (1.2s: 0.3s fade-in → 0.6s hold → 0.3s fade-out),
/// per UI/UX review recommendation (replaces the original 500ms yellow flash).
class SubtitleItem extends StatefulWidget {
  final SubtitleEntry entry;
  final bool isNewest;

  const SubtitleItem({super.key, required this.entry, this.isNewest = false});

  @override
  State<SubtitleItem> createState() => _SubtitleItemState();
}

class _SubtitleItemState extends State<SubtitleItem> with SingleTickerProviderStateMixin {
  late AnimationController _animController;
  late Animation<double> _fadeAnim;
  bool _wasCorrected = false;

  @override
  void initState() {
    super.initState();
    _animController = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 1200),
    );
    _fadeAnim = Tween<double>(begin: 0.0, end: 1.0).animate(
      CurvedAnimation(
        parent: _animController,
        curve: const Interval(0.0, 0.25, curve: Curves.easeIn),    // 0.3s fade-in
        reverseCurve: const Interval(0.75, 1.0, curve: Curves.easeOut), // 0.3s fade-out
      ),
    );
  }

  @override
  void dispose() {
    _animController.dispose();
    super.dispose();
  }

  @override
  void didUpdateWidget(SubtitleItem oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (widget.entry.isCorrected && !_wasCorrected) {
      _wasCorrected = true;
      _animController.forward().then((_) => _animController.reverse());
    }
  }

  @override
  Widget build(BuildContext context) {
    final entry = widget.entry;

    return AnimatedBuilder(
      animation: _fadeAnim,
      builder: (context, child) {
        return Container(
          margin: const EdgeInsets.only(bottom: 8),
          padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
          decoration: BoxDecoration(
            color: entry.isCorrected
                ? const Color(0xFFE3F2FD).withValues(alpha: 0.15 * _fadeAnim.value)
                : Colors.transparent,
            borderRadius: BorderRadius.circular(6),
            border: entry.isCorrected
                ? Border(
                    left: BorderSide(
                      color: const Color(0xFF42A5F5).withValues(alpha: _fadeAnim.value),
                      width: 3,
                    ),
                  )
                : null,
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // English original
              Text(
                entry.original,
                style: TextStyle(
                  color: Colors.white.withValues(alpha: 0.9),
                  fontSize: 13,
                  height: 1.4,
                ),
              ),
              const SizedBox(height: 2),
              // Chinese translation
              Row(
                children: [
                  if (entry.isCorrected)
                    const Padding(
                      padding: EdgeInsets.only(right: 4),
                      child: Icon(Icons.auto_fix_high, size: 14, color: Color(0xFF42A5F5)),
                    ),
                  Expanded(
                    child: Text(
                      entry.translation,
                      style: const TextStyle(
                        color: Color(0xFFBBDEFB),
                        fontSize: 15,
                        fontWeight: FontWeight.w500,
                        height: 1.4,
                      ),
                    ),
                  ),
                ],
              ),
              // Show old translation strike-through on correction
              if (entry.isCorrected && entry.oldTranslation != null)
                Padding(
                  padding: const EdgeInsets.only(top: 2),
                  child: Text(
                    entry.oldTranslation!,
                    style: TextStyle(
                      color: Colors.white30,
                      fontSize: 12,
                      decoration: TextDecoration.lineThrough,
                    ),
                  ),
                ),
            ],
          ),
        );
      },
    );
  }
}
