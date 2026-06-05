package audio

import (
	"testing"
)

func TestChunkValidateEmpty(t *testing.T) {
	c := Chunk{Seq: 0, Data: []byte{}}
	if err := c.Validate(); err != nil {
		t.Fatalf("empty chunk should be valid, got: %v", err)
	}
}

func TestChunkValidateTooLarge(t *testing.T) {
	c := Chunk{Seq: 0, Data: make([]byte, 3*1024*1024)} // 3MB
	if err := c.Validate(); err == nil {
		t.Fatal("3MB chunk should be rejected")
	}
}

func TestChunkValidateNormal(t *testing.T) {
	c := Chunk{Seq: 1, Data: make([]byte, 1024)}
	if err := c.Validate(); err != nil {
		t.Fatalf("normal chunk should be valid, got: %v", err)
	}
}

func TestProcessorAccumulateBeforeTrigger(t *testing.T) {
	p := NewProcessor()

	// 添加小于阈值的 chunk，不应该触发
	chunk := Chunk{Seq: 0, Data: make([]byte, 32000)} // 32000 bytes < 64000
	_, ready := p.Add(chunk)
	if ready {
		t.Fatal("should not trigger before reaching 64000 bytes")
	}

	// 再添加 32000 bytes，刚好 64000，应该触发
	chunk2 := Chunk{Seq: 1, Data: make([]byte, 32000)}
	result, ready := p.Add(chunk2)
	if !ready {
		t.Fatal("should trigger at 64000 bytes")
	}
	if len(result) != 64000 {
		t.Fatalf("expected 64000 bytes result, got %d", len(result))
	}
}

func TestProcessorSkipsEmptyChunk(t *testing.T) {
	p := NewProcessor()

	// 空 chunk 不累积
	chunk := Chunk{Seq: 0, Data: []byte{}}
	_, ready := p.Add(chunk)
	if ready {
		t.Fatal("empty chunk should not trigger")
	}

	// 缓冲区应该为空
	chunk2 := Chunk{Seq: 1, Data: make([]byte, 64000)}
	result, ready := p.Add(chunk2)
	if !ready {
		t.Fatal("should trigger")
	}
	if len(result) != 64000 {
		t.Fatalf("expected 64000 bytes, got %d", len(result))
	}
}

func TestProcessorResetsAfterTrigger(t *testing.T) {
	p := NewProcessor()

	// 触发一次
	_, _ = p.Add(Chunk{Seq: 0, Data: make([]byte, 64000)})

	// 之后缓冲区应该从零开始
	_, ready := p.Add(Chunk{Seq: 1, Data: make([]byte, 32000)})
	if ready {
		t.Fatal("buffer should be reset, 32000 bytes alone should not trigger")
	}
}

func TestProcessorOversizeTriggers(t *testing.T) {
	p := NewProcessor()

	// 一次性给 128000 bytes（2 倍阈值）
	chunk := Chunk{Seq: 0, Data: make([]byte, 128000)}
	result, ready := p.Add(chunk)
	if !ready {
		t.Fatal("oversized chunk should trigger")
	}
	if len(result) != 128000 {
		t.Fatalf("expected 128000 bytes, got %d", len(result))
	}
}
