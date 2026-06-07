package tts

import (
	"strings"
	"testing"
	"time"
)

func TestStreamChunkerWaitsOnShortSoftBoundary(t *testing.T) {
	chunker := NewStreamChunker()
	now := time.Now()

	got := chunker.Add("\u4f60\u597d\uff0c", now)
	if len(got) != 0 {
		t.Fatalf("expected short soft-boundary text to wait, got %#v", got)
	}
}

func TestStreamChunkerFlushesStrongBoundary(t *testing.T) {
	chunker := NewStreamChunker()
	now := time.Now()
	text := "\u8fd9\u662f\u4e00\u6bb5\u5df2\u7ecf\u5b8c\u6574\u7684\u4e2d\u6587\u53e5\u5b50\uff0c\u957f\u5ea6\u8db3\u591f\u8ba9\u8bed\u97f3\u8fde\u7eed\u64ad\u653e\u3002"

	got := chunker.Add(text, now)
	if len(got) != 1 || got[0] != text {
		t.Fatalf("expected one strong-boundary chunk, got %#v", got)
	}
}

func TestStreamChunkerWaitsOnShortStrongBoundary(t *testing.T) {
	chunker := NewStreamChunker()
	now := time.Now()

	got := chunker.Add("\u6211\u660e\u767d\u4e86\u3002", now)
	if len(got) != 0 {
		t.Fatalf("expected short strong-boundary text to wait, got %#v", got)
	}
}

func TestStreamChunkerFlushesSoftBoundaryAfterTargetLength(t *testing.T) {
	chunker := NewStreamChunker()
	now := time.Now()
	text := strings.Repeat("\u8fde\u7eed\u4e2d\u6587", 12) + "\uff0c"

	got := chunker.Add(text, now)
	if len(got) != 1 || got[0] != text {
		t.Fatalf("expected one soft-boundary chunk, got %#v", got)
	}
}

func TestStreamChunkerTimeoutFlushesLongText(t *testing.T) {
	chunker := NewStreamChunker()
	start := time.Now()
	text := strings.Repeat("\u6ca1\u6709\u6807\u70b9", 12)

	if got := chunker.Add(text, start); len(got) != 0 {
		t.Fatalf("expected initial text to wait, got %#v", got)
	}

	got := chunker.Add("\u7ee7\u7eed", start.Add(firstChunkMaxWait+time.Millisecond))
	if len(got) != 1 {
		t.Fatalf("expected timeout chunk, got %#v", got)
	}
}

func TestStreamChunkerForceFlushesTail(t *testing.T) {
	chunker := NewStreamChunker()
	now := time.Now()
	chunker.Add("\u5c3e\u5df4", now)

	got := chunker.Flush(now)
	if len(got) != 1 || got[0] != "\u5c3e\u5df4" {
		t.Fatalf("expected force-flushed tail, got %#v", got)
	}
}
