import 'package:flutter_test/flutter_test.dart';
import 'package:verba_app/providers/session_provider.dart';
import 'package:verba_app/models/subtitle_entry.dart';

void main() {
  group('SubtitleListNotifier 字幕列表状态测试', () {
    late SubtitleListNotifier notifier;

    setUp(() {
      notifier = SubtitleListNotifier();
    });

    test('初始状态为空列表', () {
      expect(notifier.state, isEmpty);
    });

    test('追加字幕', () {
      final entry = SubtitleEntry(
        segmentId: 0,
        original: 'First',
        translation: '第一条',
        revision: 1,
        createdAt: DateTime.now(),
      );

      notifier.addSubtitle(entry);
      expect(notifier.state.length, 1);
      expect(notifier.state[0].segmentId, 0);
      expect(notifier.state[0].original, 'First');
    });

    test('追加多条字幕，保持顺序', () {
      for (var i = 0; i < 5; i++) {
        notifier.addSubtitle(SubtitleEntry(
          segmentId: i,
          original: 'Sentence $i',
          translation: '第$i句',
          revision: 1,
          createdAt: DateTime.now(),
        ));
      }

      expect(notifier.state.length, 5);
      expect(notifier.state[0].segmentId, 0);
      expect(notifier.state[4].segmentId, 4);
    });

    test('applyCorrection 更新指定字幕', () {
      notifier.addSubtitle(SubtitleEntry(
        segmentId: 3,
        original: 'Original',
        translation: '旧译文',
        revision: 1,
        createdAt: DateTime.now(),
      ));

      notifier.applyCorrection(3, '新译文', 2);

      final entry = notifier.state[0];
      expect(entry.translation, '新译文');
      expect(entry.revision, 2);
      expect(entry.isCorrected, true);
      expect(entry.oldTranslation, '旧译文');
    });

    test('applyCorrection 使用当前完整译文作为修正前文本', () {
      notifier.addSubtitle(SubtitleEntry(
        segmentId: 3,
        original: 'There was a researcher...',
        translation: '有个研究员像电影里那样冲进房间，噢着“天哪”。',
        revision: 1,
        createdAt: DateTime.now(),
      ));

      notifier.applyCorrection(
        3,
        '有个研究员像电影里那样冲进房间，惊呼“天哪”。',
        2,
        oldText: '噢着',
      );

      final entry = notifier.state[0];
      expect(entry.translation, '有个研究员像电影里那样冲进房间，惊呼“天哪”。');
      expect(entry.oldTranslation, '有个研究员像电影里那样冲进房间，噢着“天哪”。');
    });

    test('applyCorrection 低版本号被拒绝', () {
      notifier.addSubtitle(SubtitleEntry(
        segmentId: 1,
        original: 'Text',
        translation: '当前译文',
        revision: 3, // 已经是 revision 3
        createdAt: DateTime.now(),
      ));

      notifier.applyCorrection(1, '过时的译文', 2); // revision 2 < 3

      final entry = notifier.state[0];
      expect(entry.translation, '当前译文'); // 不变
      expect(entry.revision, 3); // 不变
      expect(entry.isCorrected, false); // 不变
    });

    test('applyCorrection 不存在的 segmentId 无影响', () {
      notifier.addSubtitle(SubtitleEntry(
        segmentId: 1,
        original: 'Text',
        translation: '译文',
        revision: 1,
        createdAt: DateTime.now(),
      ));

      notifier.applyCorrection(999, '不存在', 2);

      expect(notifier.state.length, 1);
      expect(notifier.state[0].translation, '译文'); // unchanged
    });

    test('列表超过 200 条时裁剪最早的条目', () {
      for (var i = 0; i < 250; i++) {
        notifier.addSubtitle(SubtitleEntry(
          segmentId: i,
          original: 'Sentence $i',
          translation: '第$i句',
          revision: 1,
          createdAt: DateTime.now(),
        ));
      }

      expect(notifier.state.length, 200);
      // 最前面 50 条被裁剪，第一条应该是 segmentId=50
      expect(notifier.state[0].segmentId, 50);
      // 最后一条是 249
      expect(notifier.state[199].segmentId, 249);
    });

    test('clear 清空列表', () {
      for (var i = 0; i < 5; i++) {
        notifier.addSubtitle(SubtitleEntry(
          segmentId: i,
          original: 'S-$i',
          translation: 'T-$i',
          revision: 1,
          createdAt: DateTime.now(),
        ));
      }

      notifier.clear();
      expect(notifier.state, isEmpty);
    });
  });
}
