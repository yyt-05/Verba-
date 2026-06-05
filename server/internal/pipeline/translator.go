package pipeline

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type Translator struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

type TranslatorConfig struct {
	APIKey  string
	BaseURL string
	Model   string
}

func NewTranslator(cfg TranslatorConfig) *Translator {
	return &Translator{
		apiKey:  cfg.APIKey,
		baseURL: cfg.BaseURL,
		model:   cfg.Model,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

// Translate converts English text to Chinese.
func (t *Translator) Translate(englishText string) (string, error) {
	if t.apiKey == "" {
		log.Printf("[translate] demo mode text=%q", englishText)
		return "[demo] 这是演示翻译结果。", nil
	}

	start := time.Now()

	body := chatRequest{
		Model: t.model,
		Messages: []chatMessage{
			{Role: "system", Content: "你是一个专业的中英翻译助手。请将以下英文翻译成简洁准确的中文。只返回翻译结果，不要添加任何解释。"},
			{Role: "user", Content: englishText},
		},
		Temperature: 0.1,
	}

	payload, _ := json.Marshal(body)
	url := t.baseURL + "/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("translate request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+t.apiKey)
	req.Header.Set("Content-Type", "application/json")

	log.Printf("[translate] request model=%s text=%q", t.model, englishText)
	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("translate call: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("translate http %d: %s", resp.StatusCode, string(respBody))
	}

	var cr chatResponse
	if err := json.Unmarshal(respBody, &cr); err != nil {
		return "", fmt.Errorf("translate parse: %w", err)
	}
	if len(cr.Choices) == 0 {
		return "", fmt.Errorf("translate empty response")
	}

	result := cr.Choices[0].Message.Content
	log.Printf("[translate] response text=%q dur=%dms", result, time.Since(start).Milliseconds())
	return result, nil
}

// Correct calls the LLM with a pre-built correction prompt.
// Uses the same API — reuses the translator's chat completions endpoint.
func (t *Translator) Correct(prompt string) ([]CorrectionSuggestion, error) {
	if t.apiKey == "" {
		log.Printf("[corrector] demo mode promptLen=%d", len(prompt))
		return []CorrectionSuggestion{
			{SegmentIndex: 0, NewTranslation: "[demo] 修正后的翻译。", Confidence: 0.9},
		}, nil
	}

	start := time.Now()

	body := chatRequest{
		Model: t.model,
		Messages: []chatMessage{
			{Role: "user", Content: prompt},
		},
		Temperature: 0.1,
	}

	payload, _ := json.Marshal(body)
	url := t.baseURL + "/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("correct request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+t.apiKey)
	req.Header.Set("Content-Type", "application/json")

	log.Printf("[corrector] request model=%s promptLen=%d", t.model, len(prompt))
	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("correct call: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("correct http %d: %s", resp.StatusCode, string(respBody))
	}

	var cr chatResponse
	if err := json.Unmarshal(respBody, &cr); err != nil {
		return nil, fmt.Errorf("correct parse: %w", err)
	}
	if len(cr.Choices) == 0 {
		return nil, fmt.Errorf("correct empty response")
	}

	var suggestions []CorrectionSuggestion
	if err := json.Unmarshal([]byte(cr.Choices[0].Message.Content), &suggestions); err != nil {
		// If JSON parse fails, return empty — not all LLM responses are valid JSON
		log.Printf("[corrector] parse suggestions failed: %v raw=%q", err, cr.Choices[0].Message.Content)
		return nil, nil
	}

	log.Printf("[corrector] response suggestions=%d dur=%dms", len(suggestions), time.Since(start).Milliseconds())
	return suggestions, nil
}

var _ = io.Discard
