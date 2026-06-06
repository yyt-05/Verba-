package pipeline

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/verba/server/internal/session"
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

// Translate converts English text to Chinese without additional context.
func (t *Translator) Translate(englishText string) (string, error) {
	return t.TranslateWithContext(englishText, nil)
}

// TranslateWithContext converts English text to Chinese using recent finalized subtitles.
func (t *Translator) TranslateWithContext(englishText string, recent []session.Sentence) (string, error) {
	if t.apiKey == "" {
		log.Printf("[translate] demo mode text=%q context=%d", englishText, len(recent))
		return "[demo] 这是演示翻译结果。", nil
	}

	start := time.Now()
	body := chatRequest{
		Model: t.model,
		Messages: []chatMessage{
			{Role: "system", Content: translateSystemPrompt()},
			{Role: "user", Content: buildTranslatePrompt(englishText, recent)},
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

	log.Printf("[translate] request model=%s text=%q context=%d", t.model, englishText, len(recent))
	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("translate call: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
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

func translateSystemPrompt() string {
	return `你是实时英文到中文字幕翻译引擎。
要求：
- 只输出当前句的简体中文翻译。
- 参考最近上下文，保持术语、代词指代和语气一致。
- 适合屏幕字幕，简洁自然，不要逐词硬翻。
- 保留常见技术缩写，如 API、SDK、RAG、LLM。
- 不要解释，不要输出多个版本，不要添加原文没有的信息。`
}

func buildTranslatePrompt(current string, recent []session.Sentence) string {
	var b bytes.Buffer
	if len(recent) > 0 {
		b.WriteString("最近上下文（英文 -> 中文）：\n")
		for _, s := range recent {
			b.WriteString(fmt.Sprintf("S%d English: %s\n", s.Index, s.Original))
			b.WriteString(fmt.Sprintf("S%d Chinese: %s\n", s.Index, s.Translation))
		}
		b.WriteString("\n")
	}
	b.WriteString("当前待翻译英文：\n")
	b.WriteString(current)
	b.WriteString("\n\n只输出当前句中文翻译。")
	return b.String()
}

// Correct calls the LLM with a pre-built correction prompt.
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
	if resp.StatusCode != http.StatusOK {
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
		log.Printf("[corrector] parse suggestions failed: %v raw=%q", err, cr.Choices[0].Message.Content)
		return nil, nil
	}

	log.Printf("[corrector] response suggestions=%d dur=%dms", len(suggestions), time.Since(start).Milliseconds())
	return suggestions, nil
}

var _ = io.Discard
