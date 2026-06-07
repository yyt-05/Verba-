package diarization

type Transcript struct {
	Text    string
	Speaker string
}

type Config struct {
	Provider    string
	APIKey      string
	RealtimeURL string
	Model       string
	Language    string
	OnFinal     func(sessionID string, transcript Transcript)
}
