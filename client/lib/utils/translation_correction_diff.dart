enum TranslationCorrectionPartKind { unchanged, oldText, newText }

class TranslationCorrectionPart {
  final String text;
  final TranslationCorrectionPartKind kind;

  const TranslationCorrectionPart(this.text, this.kind);
}

List<TranslationCorrectionPart> buildTranslationCorrectionParts({
  required String oldText,
  required String newText,
}) {
  final oldValue = oldText.trim();
  final newValue = newText.trim();
  if (oldValue.isEmpty || newValue.isEmpty || oldValue == newValue) {
    return [
      TranslationCorrectionPart(
        newValue,
        TranslationCorrectionPartKind.unchanged,
      ),
    ];
  }

  final oldChars = oldValue.runes.toList(growable: false);
  final newChars = newValue.runes.toList(growable: false);
  if (oldChars.length * newChars.length > 40000) {
    return _prefixSuffixDiff(oldValue, newValue);
  }

  final table = List.generate(
    oldChars.length + 1,
    (_) => List<int>.filled(newChars.length + 1, 0),
  );

  for (var i = oldChars.length - 1; i >= 0; i--) {
    for (var j = newChars.length - 1; j >= 0; j--) {
      if (oldChars[i] == newChars[j]) {
        table[i][j] = table[i + 1][j + 1] + 1;
      } else {
        final skipOld = table[i + 1][j];
        final skipNew = table[i][j + 1];
        table[i][j] = skipOld > skipNew ? skipOld : skipNew;
      }
    }
  }

  final result = <TranslationCorrectionPart>[];
  var oldIndex = 0;
  var newIndex = 0;
  while (oldIndex < oldChars.length && newIndex < newChars.length) {
    if (oldChars[oldIndex] == newChars[newIndex]) {
      _appendPart(
        result,
        String.fromCharCode(newChars[newIndex]),
        TranslationCorrectionPartKind.unchanged,
      );
      oldIndex++;
      newIndex++;
    } else if (table[oldIndex + 1][newIndex] >= table[oldIndex][newIndex + 1]) {
      _appendPart(
        result,
        String.fromCharCode(oldChars[oldIndex]),
        TranslationCorrectionPartKind.oldText,
      );
      oldIndex++;
    } else {
      _appendPart(
        result,
        String.fromCharCode(newChars[newIndex]),
        TranslationCorrectionPartKind.newText,
      );
      newIndex++;
    }
  }

  while (oldIndex < oldChars.length) {
    _appendPart(
      result,
      String.fromCharCode(oldChars[oldIndex]),
      TranslationCorrectionPartKind.oldText,
    );
    oldIndex++;
  }
  while (newIndex < newChars.length) {
    _appendPart(
      result,
      String.fromCharCode(newChars[newIndex]),
      TranslationCorrectionPartKind.newText,
    );
    newIndex++;
  }

  return result;
}

List<TranslationCorrectionPart> _prefixSuffixDiff(
  String oldValue,
  String newValue,
) {
  var prefix = 0;
  final minLength = oldValue.length < newValue.length
      ? oldValue.length
      : newValue.length;
  while (prefix < minLength && oldValue[prefix] == newValue[prefix]) {
    prefix++;
  }

  var suffix = 0;
  while (suffix < minLength - prefix &&
      oldValue[oldValue.length - 1 - suffix] ==
          newValue[newValue.length - 1 - suffix]) {
    suffix++;
  }
  final oldChanged = oldValue.substring(prefix, oldValue.length - suffix);
  final newChanged = newValue.substring(prefix, newValue.length - suffix);
  final result = <TranslationCorrectionPart>[];

  if (prefix > 0) {
    result.add(
      TranslationCorrectionPart(
        newValue.substring(0, prefix),
        TranslationCorrectionPartKind.unchanged,
      ),
    );
  }
  if (oldChanged.isNotEmpty) {
    result.add(
      TranslationCorrectionPart(
        oldChanged,
        TranslationCorrectionPartKind.oldText,
      ),
    );
  }
  if (newChanged.isNotEmpty) {
    result.add(
      TranslationCorrectionPart(
        newChanged,
        TranslationCorrectionPartKind.newText,
      ),
    );
  }
  if (suffix > 0) {
    result.add(
      TranslationCorrectionPart(
        newValue.substring(newValue.length - suffix),
        TranslationCorrectionPartKind.unchanged,
      ),
    );
  }

  return result;
}

void _appendPart(
  List<TranslationCorrectionPart> parts,
  String text,
  TranslationCorrectionPartKind kind,
) {
  if (text.isEmpty) return;
  if (parts.isNotEmpty && parts.last.kind == kind) {
    final last = parts.removeLast();
    parts.add(TranslationCorrectionPart(last.text + text, kind));
    return;
  }
  parts.add(TranslationCorrectionPart(text, kind));
}
