import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:window_manager/window_manager.dart';
import 'pages/home_page.dart';
import 'theme/verba_theme.dart';

void main() async {
  WidgetsFlutterBinding.ensureInitialized();

  // Configure frameless, semi-transparent, always-on-top floating window
  await windowManager.ensureInitialized();
  await windowManager.setAsFrameless();
  await windowManager.setSize(const Size(360, 640));
  await windowManager.setAlwaysOnTop(true);
  await windowManager.setBackgroundColor(Colors.transparent);
  await windowManager.center();
  await windowManager.show();

  runApp(const ProviderScope(child: VerbaApp()));
}

class VerbaApp extends StatelessWidget {
  const VerbaApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Verba - AI 双语字幕助手',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(
          seedColor: VerbaColors.brandBlue,
          brightness: Brightness.dark,
        ),
        scaffoldBackgroundColor: Colors.transparent,
        useMaterial3: true,
      ),
      home: const HomePage(),
    );
  }
}
