package audio

import (
	"encoding/binary"
	"testing"
)

func TestWriteWAVHeader(t *testing.T) {
	// 1000 bytes of fake PCM16 data
	pcm := make([]byte, 1000)
	for i := range pcm {
		pcm[i] = byte(i % 256)
	}

	result := WriteWAVHeader(16000, pcm)

	// Header + data
	if len(result) != 44+1000 {
		t.Fatalf("expected 1044 bytes, got %d", len(result))
	}

	// Check RIFF magic
	if string(result[0:4]) != "RIFF" {
		t.Fatal("missing RIFF header")
	}
	if string(result[8:12]) != "WAVE" {
		t.Fatal("missing WAVE header")
	}

	// Check sample rate at offset 24-28
	sr := binary.LittleEndian.Uint32(result[24:28])
	if sr != 16000 {
		t.Fatalf("expected 16000Hz, got %d", sr)
	}

	// Check PCM data preserved
	for i := 0; i < 1000; i++ {
		if result[44+i] != byte(i%256) {
			t.Fatalf("PCM data corrupted at byte %d", i)
		}
	}
}

func TestWriteWAVHeader_Empty(t *testing.T) {
	result := WriteWAVHeader(48000, []byte{})
	if len(result) != 44 {
		t.Fatalf("expected 44 bytes (header only), got %d", len(result))
	}
}
