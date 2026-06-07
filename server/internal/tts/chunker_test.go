package tts

import (
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
	text := "\u8fd9\u662f\u4e00\u6bb5\u5df2\u7ecf\u5b8c\u6574\u7684\u4e2d\u6587\u53e5\u5b50\u3002"

	got := chunker.Add(text, now)
	if len(got) != 1 || got[0] != text {
		t.Fatalf("expected one strong-boundary chunk, got %#v", got)
	}
}

func TestStreamChunkerFlushesSoftBoundaryAfterTargetLength(t *testing.T) {
	chunker := NewStreamChunker()
	now := time.Now()
	text := "\u8fd9\u662f\u4e00\u6bb5\u8db3\u591f\u957f\u7684\u4e2d\u6587\u5185\u5bb9\u7528\u6765\u6d4b\u8bd5\u5f31\u8fb9\u754c\u662f\u5426\u4f1a\u5728\u8fbe\u5230\u76ee\u6807\u957f\u5ea6\u540e\u5207\u5206\uff0c"

	got := chunker.Add(text, now)
	if len(got) != 1 || got[0] != text {
		t.Fatalf("expected one soft-boundary chunk, got %#v", got)
	}
}

func TestStreamChunkerTimeoutFlushesLongText(t *testing.T) {
	chunker := NewStreamChunker()
	start := time.Now()
	text := "\u8fd9\u662f\u4e00\u6bb5\u6ca1\u6709\u6807\u70b9\u4f46\u662f\u5df2\u7ecf\u8db3\u591f\u957f\u7684\u4e2d\u6587\u5185\u5bb9\u7528\u6765\u6d4b\u8bd5\u8d85\u65f6\u5207\u5206"

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
