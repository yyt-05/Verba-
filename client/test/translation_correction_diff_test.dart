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

  test('keeps long shared subtitle context outside the correction highlight', () {
    final parts = buildTranslationCorrectionParts(
      oldText: '有个研究员像电影里那样冲进房间，噢着“天哪”。',
      newText: '有个研究员像电影里那样冲进房间，惊呼“天哪”。',
    );

    expect(parts.map((part) => '${part.kind}:${part.text}').toList(), [
      'TranslationCorrectionPartKind.unchanged:有个研究员像电影里那样冲进房间，',
      'TranslationCorrectionPartKind.oldText:噢着',
      'TranslationCorrectionPartKind.newText:惊呼',
      'TranslationCorrectionPartKind.unchanged:“天哪”。',
    ]);
    expect(shouldShowInlineCorrectionParts(parts, '有个研究员像电影里那样冲进房间，惊呼“天哪”。'), true);
  });

  test('hides inline correction when the change would repeat whole subtitles', () {
    final parts = buildTranslationCorrectionParts(
      oldText: '收藏量破千，太疯狂了。我原本很期待。',
      newText: '很久没用了，现在用起来还挺有意思的。',
    );

    expect(
      shouldShowInlineCorrectionParts(parts, '很久没用了，现在用起来还挺有意思的。'),
      false,
    );
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
