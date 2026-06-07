package audio

import (
	"encoding/binary"
	"testing"
)

func TestSegmenterWaitsForTrailingSilence(t *testing.T) {
	seg := NewSegmenter(48000)

	if out, ready := seg.AddPCM(testPCM(48000, 300, 2000)); ready || out != nil {
		t.Fatal("first voiced chunk should not complete a segment")
	}
	if out, ready := seg.AddPCM(testPCM(48000, 300, 2000)); ready || out != nil {
		t.Fatal("second voiced chunk should not complete a segment")
	}
	if out, ready := seg.AddPCM(testPCM(48000, 300, 2000)); ready || out != nil {
		t.Fatal("third voiced chunk should wait for silence")
	}
	if out, ready := seg.AddPCM(testPCM(48000, 600, 0)); ready || out != nil {
		t.Fatal("silence below threshold should not complete a segment")
	}

	out, ready := seg.AddPCM(testPCM(48000, 200, 0))
	if !ready {
		t.Fatal("trailing silence should complete the segment")
	}
	if len(out) == 0 {
		t.Fatal("completed segment should contain audio")
	}
}

func TestSegmenterDropsShortNoise(t *testing.T) {
	seg := NewSegmenter(48000)

	seg.AddPCM(testPCM(48000, 100, 2000))
	out, ready := seg.AddPCM(testPCM(48000, 800, 0))
	if ready || out != nil {
		t.Fatal("short voiced noise should be dropped")
	}
}

func TestSegmenterIgnoresLeadingSilence(t *testing.T) {
	seg := NewSegmenter(48000)

	out, ready := seg.AddPCM(testPCM(48000, 1000, 0))
	if ready || out != nil {
		t.Fatal("leading silence should not create a segment")
	}
}

func TestSegmenterForceFlushesLongSpeech(t *testing.T) {
	seg := NewSegmenter(48000)

	var ready bool
	var out []byte
	for i := 0; i < 34; i++ {
		out, ready = seg.AddPCM(testPCM(48000, 300, 2000))
		if ready {
			break
		}
	}

	if !ready {
		t.Fatal("long speech should be force-flushed")
	}
	if len(out) == 0 {
		t.Fatal("force-flushed segment should contain audio")
	}
}

func TestSegmenterFlushesLowLevelSystemAudio(t *testing.T) {
	seg := NewSegmenter(48000)

	var ready bool
	var out []byte
	for i := 0; i < 7; i++ {
		out, ready = seg.AddPCM(testPCM(48000, 300, 20))
		if ready {
			break
		}
	}

	if !ready {
		t.Fatal("low-level system audio should be flushed for ASR after about 2 seconds")
	}
	if len(out) == 0 {
		t.Fatal("completed segment should contain audio")
	}
}

func testPCM(sampleRate, durationMs int, amplitude int16) []byte {
	samples := sampleRate * durationMs / 1000
	data := make([]byte, samples*2)
	for i := 0; i < samples; i++ {
		binary.LittleEndian.PutUint16(data[i*2:i*2+2], uint16(amplitude))
	}
	return data
}
