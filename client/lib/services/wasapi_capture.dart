import 'dart:ffi';
import 'dart:typed_data';

/// FFI bindings to wasapi_capture.dll.
class WasapiCapture {
  late final DynamicLibrary _lib;
  late final int Function() _start;
  late final void Function() _stop;
  late final double Function() _getLevel;
  late final int Function() _isCapturing;
  late final Pointer<Int8> Function() _getDiag;
  late final int Function() _getLastError;
  late final int Function() _readAudio;
  late final Pointer<Uint8> Function() _getAudioData;
  late final int Function() _availableBytes;

  bool _loaded = false;

  void load() {
    if (_loaded) return;
    _lib = DynamicLibrary.open('wasapi_capture.dll');
    _start = _lib.lookupFunction<Int32 Function(), int Function()>('wasapi_start');
    _stop = _lib.lookupFunction<Void Function(), void Function()>('wasapi_stop');
    _getLevel = _lib.lookupFunction<Float Function(), double Function()>('wasapi_get_level');
    _isCapturing = _lib.lookupFunction<Int32 Function(), int Function()>('wasapi_is_capturing');
    _getDiag = _lib.lookupFunction<Pointer<Int8> Function(), Pointer<Int8> Function()>('wasapi_get_diag');
    _getLastError = _lib.lookupFunction<Int32 Function(), int Function()>('wasapi_get_last_error');
    _readAudio = _lib.lookupFunction<Int32 Function(), int Function()>('wasapi_read_audio');
    _getAudioData = _lib.lookupFunction<Pointer<Uint8> Function(), Pointer<Uint8> Function()>('wasapi_get_audio_data');
    _availableBytes = _lib.lookupFunction<Int32 Function(), int Function()>('wasapi_available_bytes');
    _loaded = true;
  }

  /// Returns 0 on success, non-zero error code on failure.
  int start() {
    if (!_loaded) load();
    return _start();
  }

  void stop() {
    if (!_loaded) return;
    _stop();
  }

  double get level {
    if (!_loaded) return -1.0;
    return _getLevel();
  }

  bool get isCapturing {
    if (!_loaded) return false;
    return _isCapturing() == 1;
  }

  /// Diagnostic string from C++ side.
  String get diag {
    if (!_loaded) return 'not loaded';
    final ptr = _getDiag();
    if (ptr == nullptr) return 'null';
    // Manually read null-terminated C string
    final codeUnits = <int>[];
    var offset = 0;
    while (true) {
      final byte = ptr[offset];
      if (byte == 0) break;
      codeUnits.add(byte);
      offset++;
    }
    return String.fromCharCodes(codeUnits);
  }

  /// Last HRESULT error code from capture loop.
  int get lastError {
    if (!_loaded) return 0;
    return _getLastError();
  }

  /// Bytes currently available in the capture ring buffer.
  int get availableBytes {
    if (!_loaded) return 0;
    return _availableBytes();
  }

  /// Read captured audio. Returns null if no data available.
  Uint8List? readAudio() {
    if (!_loaded) return null;
    final copied = _readAudio();
    if (copied <= 0) return null;
    // Read from C++ internal buffer (zero allocation)
    final ptr = _getAudioData();
    return Uint8List.fromList(List.generate(copied, (i) => ptr[i]));
  }
}
