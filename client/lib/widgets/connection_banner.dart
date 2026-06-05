import 'package:flutter/material.dart';
import '../providers/session_provider.dart';

/// Thin banner showing connection/reconnection/silent/error status.
///
/// Not an intrusive AlertDialog — a gentle overlay per UI/UX review feedback.
class ConnectionBanner extends StatelessWidget {
  final SessionState state;

  const ConnectionBanner({super.key, required this.state});

  @override
  Widget build(BuildContext context) {
    final (String message, Color color, IconData icon) = switch (state) {
      SessionState.idle => ('', Colors.transparent, Icons.check),
      SessionState.requestingPermission => ('正在请求麦克风权限...', Colors.orange, Icons.mic),
      SessionState.connecting => ('正在连接服务...', Colors.orange, Icons.wifi_find),
      SessionState.listening || SessionState.capturing => ('', Colors.transparent, Icons.check),
      SessionState.silent => ('未检测到语音，请检查音频源', Colors.orange, Icons.hearing_disabled),
      SessionState.reconnecting => ('连接中断，正在重连...', Colors.orange, Icons.wifi_off),
      SessionState.apiError => ('服务异常，请检查 API 配置', Colors.red, Icons.error_outline),
      SessionState.permissionDenied => ('麦克风权限被拒绝，请前往系统设置开启', Colors.red, Icons.block),
      SessionState.audioSourceUnavailable => ('系统音频不可用，已切换至麦克风', Colors.orange, Icons.swap_horiz),
      SessionState.stopped => ('监听已停止', Colors.white38, Icons.stop_circle_outlined),
    };

    if (message.isEmpty) return const SizedBox.shrink();

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
      color: color.withValues(alpha: 0.85),
      child: SafeArea(
        bottom: false,
        child: Row(
          children: [
            Icon(icon, size: 18, color: Colors.white),
            const SizedBox(width: 8),
            Expanded(child: Text(message, style: const TextStyle(color: Colors.white, fontSize: 13))),
          ],
        ),
      ),
    );
  }
}
