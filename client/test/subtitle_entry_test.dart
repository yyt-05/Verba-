import 'package:flutter_test/flutter_test.dart';
import 'package:verba_app/models/subtitle_entry.dart';

void main() {
  group('SubtitleEntry 数据模型测试', () {
    test('构造 + 默认值', () {
      final entry = SubtitleEntry(
        segmentId: 0,
        original: 'Hello',
        translation: '你好',
        revision: 1,
        createdAt: DateTime(2026, 6, 5),
      );

      expect(entry.segmentId, 0);
      expect(entry.original, 'Hello');
      expect(entry.translation, '你好');
      expect(entry.revision, 1);
      expect(entry.isCorrected, false);
      expect(entry.oldTranslation, isNull);
    });

    test('fromJson 解析 subtitle.final 事件', () {
      final json = {
        'segmentId': 7,
        'original': 'The cat sat.',
        'translation': '猫坐着。',
      };

      final entry = SubtitleEntry.fromJson(json);
      expect(entry.segmentId, 7);
      expect(entry.original, 'The cat sat.');
      expect(entry.translation, '猫坐着。');
      expect(entry.revision, 1); // 默认值
      expect(entry.isCorrected, false);
    });

    test('fromJson 缺失字段用默认值', () {
      final entry = SubtitleEntry.fromJson({'segmentId': 1});
      expect(entry.segmentId, 1);
      expect(entry.original, '');
      expect(entry.translation, '');
      expect(entry.revision, 1);
    });

    test('copyWith 不改变未指定字段', () {
      final original = SubtitleEntry(
        segmentId: 3,
        original: 'Hello',
        translation: '你好',
        revision: 1,
        createdAt: DateTime(2026, 6, 5),
      );

      final updated = original.copyWith(translation: '你好世界');
      expect(updated.segmentId, 3);
      expect(updated.original, 'Hello');
      expect(updated.translation, '你好世界');
      expect(updated.revision, 1); // unchanged
    });

    test('copyWith 递增 revision', () {
      final original = SubtitleEntry(
        segmentId: 3,
        original: 'Hello',
        translation: '你好',
        revision: 1,
        createdAt: DateTime.now(),
      );

      final updated = original.copyWith(
        translation: '你好世界',
        revision: 2,
        isCorrected: true,
        oldTranslation: '你好',
      );

      expect(updated.translation, '你好世界');
      expect(updated.revision, 2);
      expect(updated.isCorrected, true);
      expect(updated.oldTranslation, '你好');
    });
  });
}
