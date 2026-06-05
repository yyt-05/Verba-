import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
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
}
