import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:verba_app/models/subtitle_entry.dart';
import 'package:verba_app/providers/session_provider.dart';

void main() {
  group('parseSubtitleEventData', () {
    test('reads wrapped subtitle.final payload from server event', () {
      final raw = jsonEncode({
        'id': 1,
        'type': 'subtitle.final',
        'data': {'segmentId': 7, 'original': 'Hello', 'translation': 'Ni hao'},
        'segmentId': 7,
        'revision': 1,
      });

      final data = parseSubtitleEventData(raw);

      expect(data, isNotNull);
      expect(data!['type'], 'subtitle.final');
      expect(data['segmentId'], 7);
      expect(data['original'], 'Hello');
      expect(data['translation'], 'Ni hao');
      expect(data['revision'], 1);
    });

    test('reads wrapped subtitle.corrected payload from server event', () {
      final raw = jsonEncode({
        'id': 2,
        'type': 'subtitle.corrected',
        'data': {'segmentId': 7, 'newText': 'Corrected text', 'revision': 2},
      });

      final data = parseSubtitleEventData(raw);

      expect(data, isNotNull);
      expect(data!['type'], 'subtitle.corrected');
      expect(data['segmentId'], 7);
      expect(data['newText'], 'Corrected text');
      expect(data['revision'], 2);
    });

    test('keeps flat payloads compatible', () {
      final raw = jsonEncode({
        'type': 'subtitle.final',
        'segmentId': 3,
        'original': 'Flat',
        'translation': 'Flat translation',
      });

      final data = parseSubtitleEventData(raw);

      expect(data, isNotNull);
      expect(data!['type'], 'subtitle.final');
      expect(data['segmentId'], 3);
      expect(data['translation'], 'Flat translation');
    });

    test('returns null for malformed json', () {
      expect(parseSubtitleEventData('{bad json'), isNull);
    });
  });

  group('cleanSubtitleText', () {
    test('unwraps common text payloads before display', () {
      expect(cleanSubtitleText('{"text":"Hello world"}'), 'Hello world');
      expect(cleanSubtitleText('{"translation":"你好"}'), '你好');
    });

    test('keeps normal subtitle text unchanged', () {
      expect(cleanSubtitleText('Hello world'), 'Hello world');
    });
  });

  group('SubtitleListNotifier revisions', () {
    test('upserts draft and final for the same segment', () {
      final notifier = SubtitleListNotifier();

      notifier.upsertSubtitle(
        SubtitleEntry(
          segmentId: 1,
          original: 'The model',
          translation: '这个模型',
          revision: 1,
          createdAt: DateTime.now(),
          status: SubtitleStatus.draft,
          isFinal: false,
          eventSeq: 10,
        ),
      );
      notifier.upsertSubtitle(
        SubtitleEntry(
          segmentId: 1,
          original: 'The model works.',
          translation: '这个模型有效。',
          revision: 1,
          createdAt: DateTime.now(),
          status: SubtitleStatus.finalText,
          isFinal: true,
          eventSeq: 11,
        ),
      );

      expect(notifier.state.length, 1);
      expect(notifier.state.single.status, SubtitleStatus.finalText);
      expect(notifier.state.single.translation, '这个模型有效。');
    });

    test('ignores stale correction revisions', () {
      final notifier = SubtitleListNotifier();

      notifier.upsertSubtitle(
        SubtitleEntry(
          segmentId: 1,
          original: 'The model works.',
          translation: '这个模型有效。',
          revision: 2,
          createdAt: DateTime.now(),
        ),
      );
      notifier.applyCorrection(1, '过期译文', 1);

      expect(notifier.state.single.translation, '这个模型有效。');
      expect(notifier.state.single.revision, 2);
    });
  });
}
