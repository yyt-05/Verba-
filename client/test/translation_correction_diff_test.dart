import 'package:flutter_test/flutter_test.dart';
import 'package:verba_app/utils/translation_correction_diff.dart';

void main() {
  test('keeps shared text and only marks the changed Chinese phrase', () {
    final parts = buildTranslationCorrectionParts(
      oldText: '它充分利用了技术优势。',
      newText: '它充分发挥了技术优势。',
    );

    expect(parts.map((part) => '${part.kind}:${part.text}').toList(), [
      'TranslationCorrectionPartKind.unchanged:它充分',
      'TranslationCorrectionPartKind.oldText:利用',
      'TranslationCorrectionPartKind.newText:发挥',
      'TranslationCorrectionPartKind.unchanged:了技术优势。',
    ]);
  });

  test('does not mark identical text as a correction', () {
    final parts = buildTranslationCorrectionParts(
      oldText: '云端代码封装在简单易用的界面里。',
      newText: '云端代码封装在简单易用的界面里。',
    );

    expect(parts, hasLength(1));
    expect(parts.single.kind, TranslationCorrectionPartKind.unchanged);
    expect(parts.single.text, '云端代码封装在简单易用的界面里。');
  });
}
