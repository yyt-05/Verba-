import 'dart:async';
import 'dart:ffi';
import 'dart:io';
import 'dart:typed_data';

final class _WaveFormatEx extends Struct {
  @Uint16()
  external int wFormatTag;

  @Uint16()
  external int nChannels;

  @Uint32()
  external int nSamplesPerSec;

  @Uint32()
  external int nAvgBytesPerSec;

  @Uint16()
  external int nBlockAlign;

  @Uint16()
  external int wBitsPerSample;

  @Uint16()
  external int cbSize;
}

final class _WaveHdr extends Struct {
  external Pointer<Uint8> lpData;

  @Uint32()
  external int dwBufferLength;

  @Uint32()
  external int dwBytesRecorded;

  @IntPtr()
  external int dwUser;

  @Uint32()
  external int dwFlags;

  @Uint32()
  external int dwLoops;

  external Pointer<Void> lpNext;

  @IntPtr()
  external int reserved;
}

class _PendingWave {
  final Pointer<_WaveHdr> header;
  final Pointer<Void> data;
  final int byteCount;

  const _PendingWave(this.header, this.data, this.byteCount);
}

class TtsPcmPlayer {
  static const int _blockBytes = 1920; // 40 ms at 24 kHz, mono, s16le.
  static const int _prebufferBytes = 14400; // 300 ms.
  static const int _targetQueuedBytes = 14400; // 300 ms.
  static const int _maxInputBytes = 96000; // 2 seconds.
  static const int _waveFormatPcm = 1;
  static const int _waveMapper = 0xffffffff;
  static const int _callbackNull = 0;
  static const int _heapZeroMemory = 0x00000008;
  static const int _whdrDone = 0x00000001;

  late final DynamicLibrary _winmm = DynamicLibrary.open('winmm.dll');
  late final DynamicLibrary _kernel32 = DynamicLibrary.open('kernel32.dll');

  late final int Function() _getProcessHeap = _kernel32
      .lookupFunction<IntPtr Function(), int Function()>('GetProcessHeap');
  late final Pointer<Void> Function(int, int, int) _heapAlloc = _kernel32
      .lookupFunction<
        Pointer<Void> Function(IntPtr, Uint32, IntPtr),
        Pointer<Void> Function(int, int, int)
      >('HeapAlloc');
  late final int Function(int, int, Pointer<Void>) _heapFree = _kernel32
      .lookupFunction<
        Int32 Function(IntPtr, Uint32, Pointer<Void>),
        int Function(int, int, Pointer<Void>)
      >('HeapFree');

  late final int Function(
    Pointer<IntPtr>,
    int,
    Pointer<_WaveFormatEx>,
    int,
    int,
    int,
  )
  _waveOutOpen = _winmm
      .lookupFunction<
        Uint32 Function(
          Pointer<IntPtr>,
          Uint32,
          Pointer<_WaveFormatEx>,
          IntPtr,
          IntPtr,
          Uint32,
        ),
        int Function(
          Pointer<IntPtr>,
          int,
          Pointer<_WaveFormatEx>,
          int,
          int,
          int,
        )
      >('waveOutOpen');
  late final int Function(Pointer<Void>, Pointer<_WaveHdr>, int)
  _waveOutPrepareHeader = _winmm
      .lookupFunction<
        Uint32 Function(Pointer<Void>, Pointer<_WaveHdr>, Uint32),
        int Function(Pointer<Void>, Pointer<_WaveHdr>, int)
      >('waveOutPrepareHeader');
  late final int Function(Pointer<Void>, Pointer<_WaveHdr>, int) _waveOutWrite =
      _winmm.lookupFunction<
        Uint32 Function(Pointer<Void>, Pointer<_WaveHdr>, Uint32),
        int Function(Pointer<Void>, Pointer<_WaveHdr>, int)
      >('waveOutWrite');
  late final int Function(Pointer<Void>, Pointer<_WaveHdr>, int)
  _waveOutUnprepareHeader = _winmm
      .lookupFunction<
        Uint32 Function(Pointer<Void>, Pointer<_WaveHdr>, Uint32),
        int Function(Pointer<Void>, Pointer<_WaveHdr>, int)
      >('waveOutUnprepareHeader');
  late final int Function(Pointer<Void>) _waveOutReset = _winmm
      .lookupFunction<
        Uint32 Function(Pointer<Void>),
        int Function(Pointer<Void>)
      >('waveOutReset');
  late final int Function(Pointer<Void>) _waveOutClose = _winmm
      .lookupFunction<
        Uint32 Function(Pointer<Void>),
        int Function(Pointer<Void>)
      >('waveOutClose');

  final List<_PendingWave> _pending = [];
  final List<int> _buffer = [];
  Timer? _startupTimer;
  Timer? _pumpTimer;
  Timer? _cleanupTimer;
  DateTime? _lastInputAt;
  int _heap = 0;
  int _handle = 0;
  int _queuedBytes = 0;
  bool _started = false;

  void playPcm24kMono16(Uint8List pcm) {
    if (!Platform.isWindows) return;
    if (pcm.isEmpty) return;
    _buffer.addAll(pcm);
    _lastInputAt = DateTime.now();
    _trimInputBuffer();

    if (!_started) {
      if (_buffer.length >= _prebufferBytes) {
        _startPlayback();
      } else {
        _startupTimer ??= Timer(const Duration(milliseconds: 260), () {
          _startupTimer = null;
          if (_buffer.isNotEmpty) {
            _startPlayback();
          }
        });
      }
      return;
    }

    _fillDeviceQueue();
  }

  void _startPlayback() {
    if (_started) return;
    _ensureOpen();
    if (_handle == 0) return;
    _started = true;
    _pumpTimer ??= Timer.periodic(
      const Duration(milliseconds: 20),
      (_) => _pump(),
    );
    _fillDeviceQueue();
  }

  void _pump() {
    _collectDone();
    if (!_started) return;
    _fillDeviceQueue();
  }

  void _fillDeviceQueue() {
    while (_queuedBytes < _targetQueuedBytes && _buffer.length >= _blockBytes) {
      _submitPcm(_takeBytes(_blockBytes));
    }

    final lastInputAt = _lastInputAt;
    final inputIdle =
        lastInputAt != null &&
        DateTime.now().difference(lastInputAt) >
            const Duration(milliseconds: 160);
    if (_queuedBytes == 0 && _buffer.isNotEmpty && inputIdle) {
      _submitPcm(_takeBytes(_buffer.length));
    }
  }

  void _submitPcm(Uint8List pcm) {
    if (pcm.isEmpty || _handle == 0) return;
    final data = _allocBytes(pcm.length);
    data.cast<Uint8>().asTypedList(pcm.length).setAll(0, pcm);

    final header = _allocWaveHdr();
    header.ref
      ..lpData = data.cast<Uint8>()
      ..dwBufferLength = pcm.length
      ..dwBytesRecorded = 0
      ..dwUser = 0
      ..dwFlags = 0
      ..dwLoops = 0
      ..lpNext = Pointer<Void>.fromAddress(0)
      ..reserved = 0;

    final hwo = Pointer<Void>.fromAddress(_handle);
    var result = _waveOutPrepareHeader(hwo, header, sizeOf<_WaveHdr>());
    if (result != 0) {
      _free(header.cast<Void>());
      _free(data);
      return;
    }

    result = _waveOutWrite(hwo, header, sizeOf<_WaveHdr>());
    if (result != 0) {
      _waveOutUnprepareHeader(hwo, header, sizeOf<_WaveHdr>());
      _free(header.cast<Void>());
      _free(data);
      return;
    }

    _pending.add(_PendingWave(header, data, pcm.length));
    _queuedBytes += pcm.length;
    _cleanupTimer ??= Timer.periodic(
      const Duration(milliseconds: 250),
      (_) => _collectDone(),
    );
  }

  void dispose() {
    _startupTimer?.cancel();
    _startupTimer = null;
    _pumpTimer?.cancel();
    _pumpTimer = null;
    _buffer.clear();
    _cleanupTimer?.cancel();
    _cleanupTimer = null;
    if (_handle != 0) {
      final hwo = Pointer<Void>.fromAddress(_handle);
      _waveOutReset(hwo);
      for (final pending in _pending) {
        _waveOutUnprepareHeader(hwo, pending.header, sizeOf<_WaveHdr>());
        _free(pending.header.cast<Void>());
        _free(pending.data);
      }
      _pending.clear();
      _queuedBytes = 0;
      _waveOutClose(hwo);
      _handle = 0;
    }
    _started = false;
  }

  void _ensureOpen() {
    if (_handle != 0) return;

    final format = _allocWaveFormat();
    format.ref
      ..wFormatTag = _waveFormatPcm
      ..nChannels = 1
      ..nSamplesPerSec = 24000
      ..nAvgBytesPerSec = 24000 * 2
      ..nBlockAlign = 2
      ..wBitsPerSample = 16
      ..cbSize = 0;

    final handlePtr = _allocIntPtr();
    final result = _waveOutOpen(
      handlePtr,
      _waveMapper,
      format,
      0,
      0,
      _callbackNull,
    );
    _free(format.cast<Void>());

    if (result != 0) {
      _free(handlePtr.cast<Void>());
      return;
    }

    _handle = handlePtr.value;
    _free(handlePtr.cast<Void>());
  }

  void _collectDone() {
    if (_handle == 0) return;
    final hwo = Pointer<Void>.fromAddress(_handle);
    for (var i = _pending.length - 1; i >= 0; i--) {
      final pending = _pending[i];
      if ((pending.header.ref.dwFlags & _whdrDone) == 0) continue;
      _waveOutUnprepareHeader(hwo, pending.header, sizeOf<_WaveHdr>());
      _free(pending.header.cast<Void>());
      _free(pending.data);
      _queuedBytes -= pending.byteCount;
      if (_queuedBytes < 0) {
        _queuedBytes = 0;
      }
      _pending.removeAt(i);
    }
    if (_pending.isEmpty) {
      _cleanupTimer?.cancel();
      _cleanupTimer = null;
    }
  }

  Uint8List _takeBytes(int byteCount) {
    final n = byteCount.clamp(0, _buffer.length);
    final out = Uint8List.fromList(_buffer.sublist(0, n));
    _buffer.removeRange(0, n);
    return out;
  }

  void _trimInputBuffer() {
    if (_buffer.length <= _maxInputBytes) return;
    _buffer.removeRange(0, _buffer.length - _maxInputBytes);
  }

  Pointer<_WaveFormatEx> _allocWaveFormat() {
    return _allocBytes(sizeOf<_WaveFormatEx>()).cast<_WaveFormatEx>();
  }

  Pointer<_WaveHdr> _allocWaveHdr() {
    return _allocBytes(sizeOf<_WaveHdr>()).cast<_WaveHdr>();
  }

  Pointer<IntPtr> _allocIntPtr() {
    return _allocBytes(sizeOf<IntPtr>()).cast<IntPtr>();
  }

  Pointer<Void> _allocBytes(int byteCount) {
    _heap = _heap == 0 ? _getProcessHeap() : _heap;
    return _heapAlloc(_heap, _heapZeroMemory, byteCount);
  }

  void _free(Pointer<Void> pointer) {
    if (pointer.address == 0 || _heap == 0) return;
    _heapFree(_heap, 0, pointer);
  }
}
