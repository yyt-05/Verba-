package diarization

import (
	"encoding/binary"
	"net/url"
	"testing"
)

func TestTencentURLIncludesSpeakerEngineAndSignature(t *testing.T) {
	got, err := tencentURL(Config{
		AppID:     "1000000000",
		SecretID:  "secret-id",
		SecretKey: "secret-key",
		Model:     "16k_zh_en_speaker",
	}, "sess_test", 1000)
	if err != nil {
		t.Fatalf("tencentURL returned error: %v", err)
	}

	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("invalid URL: %v", err)
	}
	q := u.Query()
	if q.Get("engine_model_type") != "16k_zh_en_speaker" {
		t.Fatalf("expected speaker engine, got %q", q.Get("engine_model_type"))
	}
	if q.Get("signature") == "" {
		t.Fatal("expected signature")
	}
	if q.Get("voice_format") != "1" {
		t.Fatalf("expected PCM voice_format=1, got %q", q.Get("voice_format"))
	}
}

func TestTencentSpeakerLabelUsesZeroBasedSpeakerID(t *testing.T) {
	if got := tencentSpeakerLabel(0, nil); got != "A" {
		t.Fatalf("expected A, got %q", got)
	}
	if got := tencentSpeakerLabel(1, nil); got != "B" {
		t.Fatalf("expected B, got %q", got)
	}
}

func TestDownsample48kTo16kPCM16KeepsEveryThirdSample(t *testing.T) {
	pcm := make([]byte, 18)
	for i := 0; i < 9; i++ {
		binary.LittleEndian.PutUint16(pcm[i*2:i*2+2], uint16(i+1))
	}

	got := downsample48kTo16kPCM16(pcm)
	if len(got) != 6 {
		t.Fatalf("expected 3 samples, got %d bytes", len(got))
	}
	want := []int16{1, 4, 7}
	for i, expected := range want {
		if sample := pcm16Sample(got, i); sample != expected {
			t.Fatalf("sample %d: expected %d, got %d", i, expected, sample)
		}
	}
}
