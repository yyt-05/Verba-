import 'dart:typed_data';

/// Web and non-FFI fallback for the Windows-only WASAPI capture backend.
class WasapiCapture {
  void load() {}

  int start() => 1;

  void stop() {}

  double get level => -1.0;

  bool get isCapturing => false;

  String get diag => 'wasapi unavailable on this platform';

  int get lastError => 0;

  int get availableBytes => 0;

  Uint8List? readAudio() => null;
}
