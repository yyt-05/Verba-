import 'dart:typed_data';

class TtsPlaybackBuffer {
  final List<int> _bytes = [];

  int get length => _bytes.length;

  bool get isEmpty => _bytes.isEmpty;

  bool get isNotEmpty => _bytes.isNotEmpty;

  void append(Uint8List bytes) {
    if (bytes.isEmpty) return;
    _bytes.addAll(bytes);
  }

  Uint8List take(int byteCount) {
    final n = byteCount.clamp(0, _bytes.length);
    final out = Uint8List.fromList(_bytes.sublist(0, n));
    _bytes.removeRange(0, n);
    return out;
  }

  void clear() {
    _bytes.clear();
  }
}
