package pipeline

import (
	"strings"
	"testing"

	"github.com/verba/server/internal/session"
)

func TestNeedsCorrectionTooFewSentences(t *testing.T) {
	c := NewCorrector()
	mgr := session.NewManager(nil)
	s := mgr.Create("test")

	// 少于 6 句，永远不触发
	for range 4 {
		s.AppendSentence("text", "译文", "")
	}
	if c.NeedsCorrection(s) {
		t.Fatal("should not trigger with < 6 sentences")
	}
}

func TestNeedsCorrectionAtTriggerPoint(t *testing.T) {
	c := NewCorrector()
	mgr := session.NewManager(nil)
	s := mgr.Create("test")

	// 添加 6 句（seq=6, 6%3==0 → 触发）
	for range 5 {
		s.AppendSentence("text", "译文", "")
	}
	// 第 6 句
	s.AppendSentence("text6", "译文6", "")

	if !c.NeedsCorrection(s) {
		t.Fatal("should trigger when window >= 6")
	}
}

func TestNeedsCorrectionNotAtNonTriggerPoint(t *testing.T) {
	c := NewCorrector()
	mgr := session.NewManager(nil)
	s := mgr.Create("test")

	// 添加 7 句（seq=7, 7%3!=0 → 不触发）
	for range 7 {
		s.AppendSentence("text", "译文", "")
	}

	if c.NeedsCorrection(s) {
		t.Fatal("should not trigger when not at multiple of TriggerEvery")
	}
}

func TestBuildCorrectionPrompt(t *testing.T) {
	c := NewCorrector()

	window := []session.Sentence{
		{Index: 0, Original: "The cat sat.", Translation: "猫坐着。"},
		{Index: 1, Original: "It was happy.", Translation: "它很开心。"},
	}
	lookback := []session.Sentence{
		{Index: 0, Original: "The cat sat.", Translation: "猫坐着。"},
	}

	prompt := c.BuildCorrectionPrompt(window, lookback, "这是一段关于猫的对话。")

	// prompt 应包含原文
	if !strings.Contains(prompt, "The cat sat.") {
		t.Error("prompt should contain original text")
	}
	// prompt 应包含译文
	if !strings.Contains(prompt, "猫坐着。") {
		t.Error("prompt should contain translation")
	}
	// prompt 应要求 JSON 输出
	if !strings.Contains(prompt, "segment_index") {
		t.Error("prompt should ask for segment_index in JSON")
	}
	// prompt 应包含 confidence 字段要求
	if !strings.Contains(prompt, "confidence") {
		t.Error("prompt should mention confidence field")
	}
}

func TestWindowHashConsistency(t *testing.T) {
	window1 := []session.Sentence{
		{Index: 0, Original: "A", Translation: "甲"},
		{Index: 1, Original: "B", Translation: "乙"},
	}
	window2 := []session.Sentence{
		{Index: 0, Original: "A", Translation: "甲"},
		{Index: 1, Original: "B", Translation: "乙"},
	}
	window3 := []session.Sentence{
		{Index: 0, Original: "A", Translation: "丙"}, // different translation
		{Index: 1, Original: "B", Translation: "乙"},
	}

	h1 := WindowHash(window1)
	h2 := WindowHash(window2)
	h3 := WindowHash(window3)

	// 相同内容 = 相同 hash
	if h1 != h2 {
		t.Fatalf("identical windows should have same hash: %s vs %s", h1, h2)
	}

	// 不同内容 = 不同 hash
	if h1 == h3 {
		t.Fatal("different windows should have different hashes")
	}
}

func TestWindowHashEmpty(t *testing.T) {
	h := WindowHash([]session.Sentence{})
	if h == "" {
		t.Fatal("hash should not be empty for empty window")
	}
}

func TestCorrectorDefaults(t *testing.T) {
	c := NewCorrector()
	if c.WindowSize != 12 {
		t.Fatalf("expected default WindowSize 12, got %d", c.WindowSize)
	}
	if c.TriggerEvery != 3 {
		t.Fatalf("expected default TriggerEvery 3, got %d", c.TriggerEvery)
	}
	if c.LookbackCount != 6 {
		t.Fatalf("expected default LookbackCount 6, got %d", c.LookbackCount)
	}
}
