package pipeline

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
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
	Stream      bool          `json:"stream,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

type chatStreamResponse struct {
	Choices []struct {
		Delta chatMessage `json:"delta"`
	} `json:"choices"`
}

// Translate converts English text to Chinese without additional context.
func (t *Translator) Translate(englishText string) (string, error) {
	return t.TranslateWithContext(englishText, nil, "")
}

// TranslateStreamWithContext streams Chinese translation deltas when the
// provider supports OpenAI-compatible streaming, and returns the final text.
func (t *Translator) TranslateStreamWithContext(englishText string, recent []session.Sentence, background string, onDelta func(string)) (string, error) {
	if t.apiKey == "" {
		result, err := t.TranslateWithContext(englishText, recent, background)
		if err == nil && onDelta != nil {
			onDelta(result)
		}
		return result, err
	}

	start := time.Now()
	body := chatRequest{
		Model: t.model,
		Messages: []chatMessage{
			{Role: "system", Content: translateSystemPrompt(background)},
			{Role: "user", Content: buildTranslatePrompt(englishText, recent, background)},
		},
		Temperature: 0.1,
		Stream:      true,
	}

	payload, _ := json.Marshal(body)
	url := t.baseURL + "/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("translate stream request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+t.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	log.Printf("[translate] stream request model=%s text=%q context=%d", t.model, englishText, len(recent))
	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("translate stream call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("translate stream http %d: %s", resp.StatusCode, string(respBody))
	}

	var result bytes.Buffer
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}

		var chunk chatStreamResponse
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		for _, choice := range chunk.Choices {
			delta := choice.Delta.Content
			if delta == "" {
				continue
			}
			result.WriteString(delta)
			if onDelta != nil {
				onDelta(delta)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("translate stream read: %w", err)
	}

	text := result.String()
	log.Printf("[translate] stream response text=%q dur=%dms", text, time.Since(start).Milliseconds())
	if text == "" {
		return "", fmt.Errorf("translate stream empty response")
	}
	return text, nil
}

// TranslateWithContext converts English text to Chinese using recent finalized subtitles.
func (t *Translator) TranslateWithContext(englishText string, recent []session.Sentence, background string) (string, error) {
	if t.apiKey == "" {
		log.Printf("[translate] demo mode text=%q context=%d", englishText, len(recent))
		return "[demo] 这是演示翻译结果。", nil
	}

	start := time.Now()
	body := chatRequest{
		Model: t.model,
		Messages: []chatMessage{
			{Role: "system", Content: translateSystemPrompt(background)},
			{Role: "user", Content: buildTranslatePrompt(englishText, recent, background)},
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

func translateSystemPrompt(background string) string {
	base := `你是实时英文到中文字幕翻译引擎。
要求：
- 只输出当前句的简体中文翻译。
- 参考最近上下文，保持术语、代词指代和语气一致。
- 适合屏幕字幕，简洁自然，不要逐词硬翻。
- 保留常见技术缩写，如 API、SDK、RAG、LLM。
- 不要解释，不要输出多个版本，不要添加原文没有的信息。`
	if background != "" {
		base += "\n\n当前对话背景（AI 自动总结，用于指导术语和语境）：\n" + background
	}
	return base
}

func buildTranslatePrompt(current string, recent []session.Sentence, background string) string {
	var b bytes.Buffer
	if background != "" {
		b.WriteString("背景摘要：")
		b.WriteString(background)
		b.WriteString("\n\n")
	}
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

// DetectSpeaker asks the LLM to determine whether the current sentence
// belongs to the same speaker or a new one, based on content shift cues.
func (t *Translator) DetectSpeaker(currentEnglish, currentChinese string, recent []session.Sentence, currentSpeaker string) (string, error) {
	if t.apiKey == "" {
		if currentSpeaker == "" {
			return "A", nil
		}
		return currentSpeaker, nil
	}

	nextLabel := "B"
	if currentSpeaker == "" {
		currentSpeaker = "A"
		nextLabel = "A"
	} else if currentSpeaker == "A" {
		nextLabel = "B"
	} else if currentSpeaker == "B" {
		nextLabel = "C"
	} else {
		nextLabel = "A"
	}

	var b bytes.Buffer
	b.WriteString("判断当前这句话是谁说的。\n\n")
	b.WriteString(fmt.Sprintf("当前说话人: %s\n", currentSpeaker))
	b.WriteString(fmt.Sprintf("可选标签: %s (同一人) 或 %s (另一个人)\n\n", currentSpeaker, nextLabel))
	b.WriteString("规则：如果当前句的语气、立场、内容与上一句明显不同（如提问变回答、反驳、话题突变），\n")
	b.WriteString("则判断为另一个人；否则是同一人。\n\n")

	if len(recent) > 0 {
		b.WriteString("=== 最近对话 ===\n")
		start := len(recent) - 4
		if start < 0 {
			start = 0
		}
		for _, s := range recent[start:] {
			tag := ""
			if s.Speaker != "" {
				tag = fmt.Sprintf("[%s] ", s.Speaker)
			}
			b.WriteString(fmt.Sprintf("%s%s\n", tag, s.Original))
		}
		b.WriteString("\n")
	}

	b.WriteString(fmt.Sprintf("当前句: %s\n", currentEnglish))
	b.WriteString(fmt.Sprintf("翻译: %s\n\n", currentChinese))
	b.WriteString(fmt.Sprintf("只输出一个字母: %s 或 %s", currentSpeaker, nextLabel))

	start := time.Now()
	body := chatRequest{
		Model: t.model,
		Messages: []chatMessage{
			{Role: "user", Content: b.String()},
		},
		Temperature: 0.1,
	}
	payload, _ := json.Marshal(body)
	url := t.baseURL + "/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
	if err != nil {
		return currentSpeaker, fmt.Errorf("speaker detect request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+t.apiKey)
	req.Header.Set("Content-Type", "application/json")

	log.Printf("[speaker] detect request current=%s", currentSpeaker)
	resp, err := t.client.Do(req)
	if err != nil {
		return currentSpeaker, fmt.Errorf("speaker detect call: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return currentSpeaker, fmt.Errorf("speaker detect http %d: %s", resp.StatusCode, string(respBody))
	}

	var cr chatResponse
	if err := json.Unmarshal(respBody, &cr); err != nil {
		return currentSpeaker, fmt.Errorf("speaker detect parse: %w", err)
	}
	if len(cr.Choices) == 0 {
		return currentSpeaker, nil
	}

	label := strings.TrimSpace(cr.Choices[0].Message.Content)
	if len(label) > 1 {
		label = string(label[0])
	}
	if label != "A" && label != "B" && label != "C" {
		label = currentSpeaker
	}

	log.Printf("[speaker] detect response label=%s dur=%dms", label, time.Since(start).Milliseconds())
	return label, nil
}

// SummarizeBackground asks the LLM to distill the conversation's domain, key
// terms, and speaker perspective into a compact Chinese summary.
func (t *Translator) SummarizeBackground(sentences []session.Sentence, existingSummary string) (string, error) {
	if t.apiKey == "" {
		return "[demo] 这是一个关于AI/Transformer技术的对话。", nil
	}

	var b bytes.Buffer
	b.WriteString("基于以下对话内容，用中文总结：\n")
	b.WriteString("1. 核心话题与领域（如：技术/医学/法律/商业）\n")
	b.WriteString("2. 关键术语及其中文译法\n")
	b.WriteString("3. 说话人的立场或目标\n")
	b.WriteString("限制在150字以内，只输出总结文本。\n\n")
	if existingSummary != "" {
		b.WriteString("已有背景：")
		b.WriteString(existingSummary)
		b.WriteString("\n\n")
	}
	b.WriteString("=== 对话内容 ===\n")
	for _, s := range sentences {
		b.WriteString(fmt.Sprintf("S%d EN: %s\n", s.Index, s.Original))
		b.WriteString(fmt.Sprintf("S%d ZH: %s\n", s.Index, s.Translation))
	}

	start := time.Now()
	body := chatRequest{
		Model: t.model,
		Messages: []chatMessage{
			{Role: "user", Content: b.String()},
		},
		Temperature: 0.1,
	}
	payload, _ := json.Marshal(body)
	url := t.baseURL + "/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("summarize request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+t.apiKey)
	req.Header.Set("Content-Type", "application/json")

	log.Printf("[summarize] request sentences=%d existing=%d", len(sentences), len(existingSummary))
	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("summarize call: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("summarize http %d: %s", resp.StatusCode, string(respBody))
	}

	var cr chatResponse
	if err := json.Unmarshal(respBody, &cr); err != nil {
		return "", fmt.Errorf("summarize parse: %w", err)
	}
	if len(cr.Choices) == 0 {
		return "", fmt.Errorf("summarize empty response")
	}

	summary := cr.Choices[0].Message.Content
	log.Printf("[summarize] response len=%d dur=%dms", len(summary), time.Since(start).Milliseconds())
	return summary, nil
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
