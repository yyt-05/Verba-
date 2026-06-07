import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:verba_app/services/tts_playback_buffer.dart';

void main() {
  group('TtsPlaybackBuffer', () {
    test('keeps old audio when more audio is appended', () {
      final buffer = TtsPlaybackBuffer();
      final first = Uint8List.fromList(List<int>.generate(96000, (i) => i % 251));
      final second = Uint8List.fromList(List<int>.generate(96000, (i) => (i + 7) % 251));

      buffer.append(first);
      buffer.append(second);

      expect(buffer.length, first.length + second.length);
      expect(buffer.take(first.length), first);
      expect(buffer.take(second.length), second);
    });

    test('take removes only requested leading bytes', () {
      final buffer = TtsPlaybackBuffer();
      buffer.append(Uint8List.fromList([1, 2, 3, 4, 5]));

      expect(buffer.take(2), Uint8List.fromList([1, 2]));
      expect(buffer.take(3), Uint8List.fromList([3, 4, 5]));
      expect(buffer.isEmpty, isTrue);
    });
  });
}
