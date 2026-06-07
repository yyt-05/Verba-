import 'dart:async';
import 'dart:convert';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../models/subtitle_entry.dart';
import '../services/api_client.dart';
import '../services/tts_pcm_player.dart';
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
  final TtsPcmPlayer ttsPlayer;
  String? _sessionId;
  Timer? _uploadTimer;
  StreamSubscription<String>? _sseSub;
  bool _startedWasapi = false;
  bool _uploadInFlight = false;
  bool _ttsDesired = false;
  void Function(String)? onBackgroundSummary;

  SessionNotifier({
    required this.api,
    required this.subtitleList,
    required this.wasapi,
    required this.ttsPlayer,
  }) : super(SessionState.idle);

  String? get sessionId => _sessionId;

  Future<void> startListening() async {
    // Stop any in-progress session first to avoid race conditions
    if (_sessionId != null) {
      await stopListening();
    }
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
      if (_ttsDesired) {
        await api.setTtsEnabled(_sessionId!, true);
      }
      subtitleList.clear();
      onBackgroundSummary?.call('');
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
      _sessionId = null;
      state = SessionState.apiError;
    }
  }

  void _onSSEEvent(String rawData) {
    final data = parseSubtitleEventData(rawData);
    if (data == null) return;
    final type = data['type'] as String?;

    if (type == 'subtitle.final' ||
        type == 'translation.final' ||
        type == 'translation.draft') {
      final status = type == 'translation.draft'
          ? SubtitleStatus.draft
          : subtitleStatusFromJson(data['status'] as String?);
      final entry = SubtitleEntry(
        segmentId: (data['segmentId'] as int?) ?? 0,
        original: cleanSubtitleText((data['original'] as String?) ?? ''),
        translation: cleanSubtitleText((data['translation'] as String?) ?? ''),
        speaker: (data['speaker'] as String?) ?? '',
        revision: (data['revision'] as int?) ?? 1,
        createdAt: DateTime.now(),
        status: status,
        isFinal: (data['isFinal'] as bool?) ?? status != SubtitleStatus.draft,
        eventSeq: data['eventSeq'] as int? ?? data['id'] as int?,
        segmentSeq: data['segmentSeq'] as int?,
      );
      subtitleList.upsertSubtitle(entry);
    } else if (type == 'subtitle.corrected') {
      final segId = (data['segmentId'] as int?) ?? 0;
      final newText = cleanSubtitleText((data['newText'] as String?) ?? '');
      final rev = (data['revision'] as int?) ?? 1;
      final oldText = cleanSubtitleText((data['oldText'] as String?) ?? '');
      subtitleList.applyCorrection(segId, newText, rev, oldText: oldText);
    } else if (type == 'tts.audio.delta') {
      final audio = data['audio'] as String?;
      if (audio == null || audio.isEmpty) return;
      try {
        ttsPlayer.playPcm24kMono16(base64Decode(audio));
      } catch (_) {}
    } else if (type == 'tts.audio.reset') {
      ttsPlayer.dispose();
    } else if (type == 'background.summary') {
      final summary = (data['summary'] as String?) ?? '';
      if (summary.isNotEmpty) {
        onBackgroundSummary?.call(summary);
      }
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
    final sid = _sessionId;
    _sessionId = null;
    _uploadTimer?.cancel();
    _uploadTimer = null;
    _uploadInFlight = false;
    _sseSub?.cancel();
    _sseSub = null;
    if (_startedWasapi) {
      wasapi.stop();
      _startedWasapi = false;
    }
    ttsPlayer.dispose();

    if (sid == null) return;
    try {
      await api.stopSession(sid);
    } catch (_) {}
    state = SessionState.stopped;
  }

  Future<bool> setTtsEnabled(bool enabled) async {
    final sessionId = _sessionId;
    if (sessionId == null) {
      _ttsDesired = enabled;
      return false;
    }
    try {
      await api.setTtsEnabled(sessionId, enabled);
      _ttsDesired = enabled;
      return true;
    } catch (_) {
      return false;
    }
  }

  @override
  void dispose() {
    _uploadTimer?.cancel();
    _sseSub?.cancel();
    if (_startedWasapi) {
      wasapi.stop();
      _startedWasapi = false;
    }
    ttsPlayer.dispose();
    super.dispose();
  }
}

Map<String, dynamic>? parseSubtitleEventData(String rawData) {
  try {
    final event = jsonDecode(rawData) as Map<String, dynamic>;
    final type = event['type'] as String?;
    final nested = event['data'];

    if (nested is Map<String, dynamic>) {
      final merged = {...event, ...nested};
      if (type != null) merged['type'] = type;
      return merged;
    }

    if (nested is String) {
      final decoded = jsonDecode(nested) as Map<String, dynamic>;
      final merged = {...event, ...decoded};
      if (type != null) merged['type'] = type;
      return merged;
    }

    return event;
  } catch (_) {
    return null;
  }
}

String cleanSubtitleText(String value) {
  final trimmed = value.trim();
  if (trimmed.isEmpty) return '';

  try {
    final decoded = jsonDecode(trimmed);
    if (decoded is Map<String, dynamic>) {
      for (final key in const ['text', 'original', 'translation', 'newText']) {
        final nested = decoded[key];
        if (nested is String && nested.trim().isNotEmpty) {
          return nested.trim();
        }
      }
    }
  } catch (_) {}

  return trimmed;
}

class SubtitleListNotifier extends StateNotifier<List<SubtitleEntry>> {
  SubtitleListNotifier() : super([]);

  void addSubtitle(SubtitleEntry entry) {
    upsertSubtitle(entry);
  }

  void upsertSubtitle(SubtitleEntry entry) {
    final index = state.indexWhere((e) => e.segmentId == entry.segmentId);
    if (index >= 0) {
      final current = state[index];
      if (!_shouldApply(current, entry)) return;

      final next = [...state];
      next[index] = current.copyWith(
        original: entry.original.isNotEmpty ? entry.original : current.original,
        translation: entry.translation.isNotEmpty
            ? entry.translation
            : current.translation,
        revision: entry.revision,
        status: entry.status,
        isFinal: entry.isFinal,
        eventSeq: entry.eventSeq,
        segmentSeq: entry.segmentSeq,
      );
      state = next;
      return;
    }

    state = [...state, entry];
    if (state.length > 200) {
      state = state.sublist(state.length - 200);
    }
  }

  bool _shouldApply(SubtitleEntry current, SubtitleEntry incoming) {
    if (incoming.revision > current.revision) return true;
    if (incoming.revision < current.revision) return false;
    if ((incoming.eventSeq ?? -1) > (current.eventSeq ?? -1)) return true;
    if (current.status == SubtitleStatus.draft &&
        incoming.status != SubtitleStatus.draft) {
      return true;
    }
    return false;
  }

  void applyCorrection(
    int segmentId,
    String newTranslation,
    int revision, {
    String? oldText,
  }) {
    state = state.map((e) {
      if (e.segmentId == segmentId && revision > e.revision) {
        return e.copyWith(
          translation: newTranslation,
          revision: revision,
          status: SubtitleStatus.corrected,
          isFinal: true,
          isCorrected: true,
          oldTranslation: oldText?.isNotEmpty == true ? oldText : e.translation,
        );
      }
      return e;
    }).toList();
  }

  void clear() => state = [];
}

/// Providers

final backgroundSummaryProvider = StateProvider<String>((ref) => '');

final wasapiProvider = Provider<WasapiCapture>((ref) => WasapiCapture());

final ttsPcmPlayerProvider = Provider<TtsPcmPlayer>((ref) {
  final player = TtsPcmPlayer();
  ref.onDispose(player.dispose);
  return player;
});

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
  final notifier = SessionNotifier(
    api: ref.watch(apiClientProvider),
    subtitleList: ref.watch(subtitleListProvider.notifier),
    wasapi: ref.watch(wasapiProvider),
    ttsPlayer: ref.watch(ttsPcmPlayerProvider),
  );
  notifier.onBackgroundSummary = (summary) {
    ref.read(backgroundSummaryProvider.notifier).state = summary;
  };
  return notifier;
});
