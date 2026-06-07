package diarization

type Transcript struct {
	Text    string
	Speaker string
}

type AudioFallback struct {
	PCM []byte
}

type Config struct {
	Provider    string
	APIKey      string
	RealtimeURL string
	Model       string
	Language    string
	AppID       string
	SecretID    string
	SecretKey   string
	OnFinal     func(sessionID string, transcript Transcript)
	OnFallback  func(sessionID string, fallback AudioFallback)
}
