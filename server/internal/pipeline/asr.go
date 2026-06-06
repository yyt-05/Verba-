package pipeline

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

type ASRClient struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

type ASRConfig struct {
	APIKey  string
	BaseURL string
	Model   string
}

func NewASRClient(cfg ASRConfig) *ASRClient {
	return &ASRClient{
		apiKey:  cfg.APIKey,
		baseURL: cfg.BaseURL,
		model:   cfg.Model,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (a *ASRClient) Transcribe(audioData []byte) (string, error) {
	// Demo mode — no API key configured
	if a.apiKey == "" {
		log.Printf("[asr] demo mode bytes=%d", len(audioData))
		return "[demo] This is a placeholder transcription.", nil
	}

	start := time.Now()

	// Build multipart form
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	// file field
	part, err := w.CreateFormFile("file", "audio.wav")
	if err != nil {
		return "", fmt.Errorf("asr form file: %w", err)
	}
	if _, err := part.Write(audioData); err != nil {
		return "", fmt.Errorf("asr write audio: %w", err)
	}

	// model field
	if err := w.WriteField("model", a.model); err != nil {
		return "", fmt.Errorf("asr form model: %w", err)
	}
	// response_format
	if err := w.WriteField("response_format", "text"); err != nil {
		return "", fmt.Errorf("asr form format: %w", err)
	}
	w.Close()

	url := a.baseURL + "/audio/transcriptions"
	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return "", fmt.Errorf("asr request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Content-Type", w.FormDataContentType())

	log.Printf("[asr] request model=%s bytes=%d", a.model, len(audioData))
	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("asr call: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("asr http %d: %s", resp.StatusCode, string(body))
	}

	result := string(body)
	result = parseTranscriptionText(result)
	log.Printf("[asr] response text=%q dur=%dms", result, time.Since(start).Milliseconds())
	return result, nil
}

func parseTranscriptionText(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	var payload struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err == nil {
		return strings.TrimSpace(payload.Text)
	}

	return raw
}

var _ = io.Discard
var _ = json.Marshal
