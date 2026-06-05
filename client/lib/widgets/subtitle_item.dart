import 'package:flutter/material.dart';
import '../models/subtitle_entry.dart';
import '../theme/verba_theme.dart';

/// Single subtitle entry: English original (white) + Chinese translation (highlighted).
///
/// On correction: yellow accent flash animation (1.2s: 0.3s fade-in → 0.6s hold → 0.3s fade-out).
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
                ? VerbaColors.accentYellow.withValues(alpha: 0.16 * _fadeAnim.value)
                : Colors.transparent,
            borderRadius: BorderRadius.circular(VerbaTheme.panelRadius),
            border: entry.isCorrected
                ? Border(
                    left: BorderSide(
                      color: VerbaColors.accentYellow.withValues(alpha: _fadeAnim.value),
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
                  fontWeight: FontWeight.w600,
                  height: 1.35,
                ),
              ),
              const SizedBox(height: 3),
              // Chinese translation
              Row(
                children: [
                  if (entry.isCorrected)
                    const Padding(
                      padding: EdgeInsets.only(right: 4),
                      child: Icon(
                        Icons.auto_fix_high,
                        size: 14,
                        color: VerbaColors.accentYellow,
                      ),
                    ),
                  Expanded(
                    child: Text(
                      entry.translation,
                      style: const TextStyle(
                        color: VerbaColors.textBlue,
                        fontSize: 16,
                        fontWeight: FontWeight.w700,
                        height: 1.45,
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
                      color: VerbaColors.mutedGray.withValues(alpha: 0.55),
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
