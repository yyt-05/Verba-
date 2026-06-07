package pipeline

import (
	"testing"

	"github.com/verba/server/internal/session"
)

func TestSanitizeCurrentTranslation_RemovesRepeatedContextPrefix(t *testing.T) {
	recent := []session.Sentence{
		{Index: 0, Translation: "收藏量破千，太疯狂了。"},
		{Index: 1, Translation: "我原本很期待。"},
	}

	got := sanitizeCurrentTranslation(
		"收藏量破千，太疯狂了。我原本很期待。很久没用了，现在用起来还挺有意思的。",
		recent,
	)

	if got != "很久没用了，现在用起来还挺有意思的。" {
		t.Fatalf("expected current sentence only, got %q", got)
	}
}

func TestSanitizeCurrentTranslation_KeepsNormalTranslation(t *testing.T) {
	recent := []session.Sentence{
		{Index: 0, Translation: "上一句。"},
	}

	got := sanitizeCurrentTranslation("当前句。", recent)
	if got != "当前句。" {
		t.Fatalf("expected unchanged translation, got %q", got)
	}
}
