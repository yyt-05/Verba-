package pipeline

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"strings"

	"github.com/verba/server/internal/session"
)

// Corrector implements the sliding-window correction engine.
type Corrector struct {
	WindowSize    int // context sentences to consider (default 12)
	TriggerEvery  int // trigger correction every N new sentences (default 3)
	LookbackCount int // how many previous sentences to re-evaluate (default 6)
}

func NewCorrector() *Corrector {
	return &Corrector{
		WindowSize:    12,
		TriggerEvery:  3,
		LookbackCount: 6,
	}
}

// CorrectionSuggestion is returned by the LLM for a sentence that needs fixing.
type CorrectionSuggestion struct {
	SegmentIndex   int     `json:"segment_index"`
	NewTranslation string  `json:"new_translation"`
	Confidence     float64 `json:"confidence"`
}

// NeedsCorrection checks whether correction should fire for the current session state.
func (c *Corrector) NeedsCorrection(sess *session.Session) bool {
	count := len(sess.GetWindow(c.WindowSize * 2))
	if count < 6 {
		return false
	}
	return count >= c.TriggerEvery && (count%c.TriggerEvery) == 0
}

// BuildCorrectionPrompt creates the LLM prompt for the sliding window.
func (c *Corrector) BuildCorrectionPrompt(window []session.Sentence, lookback []session.Sentence, background string) string {
	var sb strings.Builder
	sb.WriteString("你是一个翻译校对助手。以下是最近一段内容的翻译。请检查前面几句的翻译是否准确。")
	sb.WriteString("如果发现翻译错误或不一致，请给出修正。\n\n")
	if background != "" {
		sb.WriteString("=== 对话背景 ===\n")
		sb.WriteString(background)
		sb.WriteString("\n\n")
	}
	sb.WriteString("=== 上下文原文与译文 ===\n")

	for _, s := range window {
		sb.WriteString(s.Original)
		sb.WriteString("\n译文: ")
		sb.WriteString(s.Translation)
		sb.WriteString("\n\n")
	}

	sb.WriteString("=== 请检查以下句子的翻译 ===\n")
	for _, s := range lookback {
		sb.WriteString(s.Original)
		sb.WriteString("\n当前译文: ")
		sb.WriteString(s.Translation)
		sb.WriteString("\n\n")
	}

	sb.WriteString("输出 JSON 数组: [{\"segment_index\": N, \"new_translation\": \"...\", \"confidence\": 0.0-1.0}]")
	sb.WriteString("\n只输出需要修正的句子，不需要修正的不输出。")
	return sb.String()
}

// WindowHash computes a hash of the current window for optimistic locking.
func WindowHash(window []session.Sentence) string {
	data, _ := json.Marshal(window)
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])[:16]
}

var _ = log.Println
var _ = json.Marshal
