import 'dart:convert';
import 'package:http/http.dart' as http;

/// API client for Verba backend REST endpoints.
class ApiClient {
  final String baseUrl;
  final http.Client _client = http.Client();

  ApiClient({required this.baseUrl});

  /// POST /api/v1/sessions 鈥?create a new session, returns sessionId.
  Future<String> createSession() async {
    final resp = await _client.post(Uri.parse('$baseUrl/api/v1/sessions'));
    if (resp.statusCode != 201) {
      throw ApiException('Failed to create session: ${resp.statusCode}');
    }
    final data = jsonDecode(resp.body) as Map<String, dynamic>;
    return data['session_id'] as String;
  }

  /// POST /api/v1/sessions/{id}/audio 鈥?upload an audio chunk.
  Future<void> uploadAudio(String sessionId, List<int> audioData) async {
    final resp = await _client.post(
      Uri.parse('$baseUrl/api/v1/sessions/$sessionId/audio'),
      body: audioData,
      headers: {'Content-Type': 'application/octet-stream'},
    );
    if (resp.statusCode != 202) {
      throw ApiException('Audio upload failed: ${resp.statusCode}');
    }
  }

  /// POST /api/v1/sessions/{id}/stop 鈥?end the session.
  Future<void> stopSession(String sessionId) async {
    final resp = await _client.post(
      Uri.parse('$baseUrl/api/v1/sessions/$sessionId/stop'),
    );
    if (resp.statusCode != 200) {
      throw ApiException('Stop failed: ${resp.statusCode}');
    }
  }

  Future<void> setTtsEnabled(String sessionId, bool enabled) async {
    final resp = await _client.post(
      Uri.parse('$baseUrl/api/v1/sessions/$sessionId/tts'),
      body: jsonEncode({'enabled': enabled}),
      headers: {'Content-Type': 'application/json'},
    );
    if (resp.statusCode != 200) {
      throw ApiException('TTS toggle failed: ${resp.statusCode} ${resp.body}');
    }
  }

  /// GET /api/v1/sessions/{id}/events 鈥?SSE stream.
  /// Returns a stream of raw SSE event strings.
  Stream<String> sseStream(String sessionId) async* {
    final req = http.Request(
      'GET',
      Uri.parse('$baseUrl/api/v1/sessions/$sessionId/events'),
    );
    req.headers['Accept'] = 'text/event-stream';
    req.headers['Cache-Control'] = 'no-cache';

    final resp = await _client.send(req);
    if (resp.statusCode != 200) {
      throw ApiException('SSE connection failed: ${resp.statusCode}');
    }

    final stream = resp.stream
        .transform(utf8.decoder)
        .transform(const LineSplitter());
    final dataBuffer = StringBuffer();

    await for (final line in stream) {
      if (line.startsWith('event: ')) {
        // Phase 1: dispatch based on event type (subtitle.final / subtitle.corrected / ...)
      } else if (line.startsWith('id: ')) {
        // Phase 1: track last-event-id for reconnection
      } else if (line.startsWith('data: ')) {
        dataBuffer.write(line.substring(6));
      } else if (line.isEmpty) {
        if (dataBuffer.isNotEmpty) {
          yield dataBuffer.toString();
        }
        dataBuffer.clear();
      }
    }
  }

  void dispose() => _client.close();
}

class ApiException implements Exception {
  final String message;
  const ApiException(this.message);
  @override
  String toString() => 'ApiException: $message';
}
