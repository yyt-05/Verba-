package tts

import "strings"

func SplitText(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	var chunks []string
	var b strings.Builder
	runeCount := 0
	for _, r := range text {
		b.WriteRune(r)
		runeCount++
		if isBoundary(r) || runeCount >= 18 {
			chunk := strings.TrimSpace(b.String())
			if chunk != "" {
				chunks = append(chunks, chunk)
			}
			b.Reset()
			runeCount = 0
		}
	}
	if tail := strings.TrimSpace(b.String()); tail != "" {
		chunks = append(chunks, tail)
	}
	return chunks
}

func isBoundary(r rune) bool {
	switch r {
	case '，', '。', '！', '？', '；', '：', '、', ',', '.', '!', '?', ';', ':':
		return true
	default:
		return false
	}
}
