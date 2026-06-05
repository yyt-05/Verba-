import 'dart:async';
import 'dart:convert';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../models/subtitle_entry.dart';
import '../services/api_client.dart';
import '../services/wasapi_capture.dart';

enum SessionState {
  idle,
  requestingPermission,
  capturing,
  connecting,
  listening,
  silent,
  reconnecting,
  apiError,
  permissionDenied,
  audioSourceUnavailable,
  stopped,
}

class SessionNotifier extends StateNotifier<SessionState> {
  final ApiClient api;
  final SubtitleListNotifier subtitleList;
  final WasapiCapture wasapi;
  String? _sessionId;
  Timer? _uploadTimer;
  StreamSubscription<String>? _sseSub;
  bool _startedWasapi = false;
  bool _uploadInFlight = false;

  SessionNotifier({
    required this.api,
    required this.subtitleList,
    required this.wasapi,
  }) : super(SessionState.idle);

  String? get sessionId => _sessionId;

  Future<void> startListening() async {
    state = SessionState.connecting;
    try {
      if (!wasapi.isCapturing) {
        final result = wasapi.start();
        if (result != 0) {
          state = SessionState.audioSourceUnavailable;
          return;
        }
        _startedWasapi = true;
      }

      _sessionId = await api.createSession();
      state = SessionState.listening;

      _sseSub?.cancel();
      _sseSub = api
          .sseStream(_sessionId!)
          .listen(
            _onSSEEvent,
            onError: (_) => state = SessionState.reconnecting,
          );

      // Stream small PCM chunks; the backend decides sentence boundaries.
      _uploadTimer = Timer.periodic(const Duration(milliseconds: 300), (_) {
        _uploadAudio();
      });
    } catch (e) {
      if (_startedWasapi) {
        wasapi.stop();
        _startedWasapi = false;
      }
      state = SessionState.apiError;
    }
  }

  void _onSSEEvent(String rawData) {
    final data = parseSubtitleEventData(rawData);
    if (data == null) return;
    final type = data['type'] as String?;

    if (type == 'subtitle.final') {
      final entry = SubtitleEntry(
        segmentId: (data['segmentId'] as int?) ?? 0,
        original: (data['original'] as String?) ?? '',
        translation: (data['translation'] as String?) ?? '',
        revision: (data['revision'] as int?) ?? 1,
        createdAt: DateTime.now(),
      );
      subtitleList.addSubtitle(entry);
    } else if (type == 'subtitle.corrected') {
      final segId = (data['segmentId'] as int?) ?? 0;
      final newText = (data['newText'] as String?) ?? '';
      final rev = (data['revision'] as int?) ?? 1;
      subtitleList.applyCorrection(segId, newText, rev);
    }
  }

  void _uploadAudio() {
    if (_sessionId == null || _uploadInFlight) return;
    if (wasapi.isCapturing && wasapi.availableBytes > 0) {
      final audio = wasapi.readAudio();
      if (audio != null && audio.isNotEmpty) {
        _uploadInFlight = true;
        api
            .uploadAudio(_sessionId!, audio)
            .catchError((_) {})
            .whenComplete(() => _uploadInFlight = false);
      }
    }
  }

  Future<void> stopListening() async {
    _uploadTimer?.cancel();
    _uploadTimer = null;
    _uploadInFlight = false;
    _sseSub?.cancel();
    _sseSub = null;
    if (_startedWasapi) {
      wasapi.stop();
      _startedWasapi = false;
    }

    if (_sessionId == null) return;
    try {
      await api.stopSession(_sessionId!);
    } catch (_) {}
    _sessionId = null;
    state = SessionState.stopped;
  }

  @override
  void dispose() {
    _uploadTimer?.cancel();
    _sseSub?.cancel();
    if (_startedWasapi) {
      wasapi.stop();
      _startedWasapi = false;
    }
    super.dispose();
  }
}

Map<String, dynamic>? parseSubtitleEventData(String rawData) {
  try {
    final event = jsonDecode(rawData) as Map<String, dynamic>;
    final type = event['type'] as String?;
    final nested = event['data'];

    if (nested is Map<String, dynamic>) {
      return {...event, ...nested, if (type != null) 'type': type};
    }

    if (nested is String) {
      final decoded = jsonDecode(nested) as Map<String, dynamic>;
      return {...event, ...decoded, if (type != null) 'type': type};
    }

    return event;
  } catch (_) {
    return null;
  }
}

class SubtitleListNotifier extends StateNotifier<List<SubtitleEntry>> {
  SubtitleListNotifier() : super([]);

  void addSubtitle(SubtitleEntry entry) {
    state = [...state, entry];
    if (state.length > 200) {
      state = state.sublist(state.length - 200);
    }
  }

  void applyCorrection(int segmentId, String newTranslation, int revision) {
    state = state.map((e) {
      if (e.segmentId == segmentId && revision > e.revision) {
        return e.copyWith(
          translation: newTranslation,
          revision: revision,
          isCorrected: true,
          oldTranslation: e.translation,
        );
      }
      return e;
    }).toList();
  }

  void clear() => state = [];
}

/// Providers
final wasapiProvider = Provider<WasapiCapture>((ref) => WasapiCapture());

final apiClientProvider = Provider<ApiClient>((ref) {
  return ApiClient(baseUrl: 'http://localhost:8080');
});

final subtitleListProvider =
    StateNotifierProvider<SubtitleListNotifier, List<SubtitleEntry>>((ref) {
      return SubtitleListNotifier();
    });

final sessionProvider = StateNotifierProvider<SessionNotifier, SessionState>((
  ref,
) {
  return SessionNotifier(
    api: ref.watch(apiClientProvider),
    subtitleList: ref.watch(subtitleListProvider.notifier),
    wasapi: ref.watch(wasapiProvider),
  );
});
