/// SubtitleEntry represents a single bilingual subtitle segment.
///
/// Each entry has a unique [segmentId] for targeted corrections, and a
/// [revision] number that increments with each correction (optimistic locking).
class SubtitleEntry {
  final int segmentId;
  final String original;       // English ASR text
  final String translation;     // Chinese translation
  final int revision;           // starts at 1, incremented per correction
  final DateTime createdAt;
  final bool isCorrected;       // true if this entry has been corrected at least once
  final String? oldTranslation; // previous translation before correction

  const SubtitleEntry({
    required this.segmentId,
    required this.original,
    required this.translation,
    required this.revision,
    required this.createdAt,
    this.isCorrected = false,
    this.oldTranslation,
  });

  SubtitleEntry copyWith({
    int? segmentId,
    String? original,
    String? translation,
    int? revision,
    DateTime? createdAt,
    bool? isCorrected,
    String? oldTranslation,
  }) {
    return SubtitleEntry(
      segmentId: segmentId ?? this.segmentId,
      original: original ?? this.original,
      translation: translation ?? this.translation,
      revision: revision ?? this.revision,
      createdAt: createdAt ?? this.createdAt,
      isCorrected: isCorrected ?? this.isCorrected,
      oldTranslation: oldTranslation ?? this.oldTranslation,
    );
  }

  factory SubtitleEntry.fromJson(Map<String, dynamic> json) {
    return SubtitleEntry(
      segmentId: json['segmentId'] as int,
      original: json['original'] as String? ?? '',
      translation: json['translation'] as String? ?? '',
      revision: json['revision'] as int? ?? 1,
      createdAt: DateTime.now(),
    );
  }
}
