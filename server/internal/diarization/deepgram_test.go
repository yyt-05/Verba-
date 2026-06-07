package diarization

import (
	"net/url"
	"testing"
)

func TestDeepgramURLIncludesStreamingDiarizationParams(t *testing.T) {
	got, err := deepgramURL(Config{
		RealtimeURL: "wss://api.deepgram.com/v1/listen",
		Model:       "nova-3",
		Language:    "en",
	})
	if err != nil {
		t.Fatalf("deepgramURL returned error: %v", err)
	}

	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("invalid URL: %v", err)
	}
	q := u.Query()
	if q.Get("diarize") != "true" {
		t.Fatalf("expected diarize=true, got %q", q.Get("diarize"))
	}
	if q.Get("encoding") != "linear16" {
		t.Fatalf("expected linear16 encoding, got %q", q.Get("encoding"))
	}
	if q.Get("sample_rate") != "48000" {
		t.Fatalf("expected 48000 sample_rate, got %q", q.Get("sample_rate"))
	}
}

func TestSpeakerLabelFromWordsUsesDominantSpeaker(t *testing.T) {
	words := []struct {
		Speaker int `json:"speaker"`
	}{
		{Speaker: 1},
		{Speaker: 1},
		{Speaker: 0},
	}

	if got := speakerLabelFromWords(words); got != "B" {
		t.Fatalf("expected B, got %q", got)
	}
}
