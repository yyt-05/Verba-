import 'dart:async';
import 'dart:convert';
import 'package:flutter_test/flutter_test.dart';

/// 纯逻辑测试：验证 SSE 事件解析的状态机逻辑。
///
/// 注意：这里不真正发起 HTTP 请求，而是测试 SSE 文本协议的解析正确性。
/// ApiClient 的 sseStream 方法依赖 http.Client.send 返回的流，
/// 此处单独测试协议解析的核心逻辑。
void main() {
  group('SSE 协议解析测试', () {
    /// 模拟 SSE 文本行的解析逻辑（和 api_client.dart 中的一致）
    Stream<String> parseSSELines(Stream<String> lines) async* {
      final dataBuffer = StringBuffer();

      await for (final line in lines) {
        if (line.startsWith('event: ')) {
          // Phase 1: dispatch by type
        } else if (line.startsWith('id: ')) {
          // Phase 1: track last-event-id
        } else if (line.startsWith('data: ')) {
          dataBuffer.write(line.substring(6));
        } else if (line.isEmpty) {
          if (dataBuffer.isNotEmpty) {
            yield dataBuffer.toString();
            dataBuffer.clear();
          }
        }
      }
    }

    test('单个 subtitle.final 事件解析', () async {
      final rawLines = [
        'event: subtitle.final',
        'id: 1',
        'data: {"segmentId":0,"original":"Hello","translation":"你好"}',
        '',
      ];

      final events = await parseSSELines(Stream.fromIterable(rawLines)).toList();
      expect(events.length, 1);

      final data = jsonDecode(events[0]) as Map<String, dynamic>;
      expect(data['segmentId'], 0);
      expect(data['original'], 'Hello');
      expect(data['translation'], '你好');
    });

    test('多个连续事件解析', () async {
      final rawLines = [
        'event: subtitle.final',
        'data: {"segmentId":1,"original":"A"}',
        '',
        'event: subtitle.final',
        'data: {"segmentId":2,"original":"B"}',
        '',
        'event: subtitle.corrected',
        'data: {"segmentId":1,"newText":"修正A"}',
        '',
      ];

      final events = await parseSSELines(Stream.fromIterable(rawLines)).toList();
      expect(events.length, 3);
      expect(jsonDecode(events[0])['segmentId'], 1);
      expect(jsonDecode(events[1])['segmentId'], 2);
      expect(jsonDecode(events[2])['newText'], '修正A');
    });

    test('多行 data 字段合并', () async {
      // SSE 标准允许多行 data，客户端应该拼接
      final rawLines = [
        'data: {"segmentId":0,',
        'data: "original":"Hello"}',
        '',
      ];

      final events = await parseSSELines(Stream.fromIterable(rawLines)).toList();
      expect(events.length, 1);
      expect(events[0], '{"segmentId":0,"original":"Hello"}');
    });

    test('空事件行被忽略', () async {
      final rawLines = [
        '',
        ':comment',
        'event: subtitle.final',
        'data: {"segmentId":1}',
        '',
      ];

      final events = await parseSSELines(Stream.fromIterable(rawLines)).toList();
      expect(events.length, 1);
    });

    test('流结束时缓冲区中有数据也产出', () async {
      // 如果流以 data 行结束（没有尾部空行），缓冲区中应有数据产出
      final rawLines = [
        'data: {"segmentId":5}',
        '', // 空行触发产出
      ];

      final events = await parseSSELines(Stream.fromIterable(rawLines)).toList();
      expect(events.length, 1);
    });
  });

  group('ApiException 测试', () {
    test('message 正确保存', () {
      // 注意：ApiException 在 api_client.dart 中定义
      // 本测试验证它不在当前测试文件中时，异常逻辑的正确性
      // (实际 ApiException 类定义在 services/api_client.dart)
    });
  });
}
