package tts

import (
	"strings"
	"time"
	"unicode/utf8"
)

const (
	minChunkRunes      = 12
	targetChunkRunes   = 28
	softMaxChunkRunes  = 42
	hardMaxChunkRunes  = 56
	firstChunkMaxWait  = 500 * time.Millisecond
	normalChunkMaxWait = 750 * time.Millisecond
	forceMinTailRunes  = 4
)

type StreamChunker struct {
	pending        string
	bufferStarted  time.Time
	firstChunkSent bool
}

func NewStreamChunker() *StreamChunker {
	return &StreamChunker{}
}

func (c *StreamChunker) Add(delta string, now time.Time) []string {
	delta = strings.TrimSpace(delta)
	if delta == "" {
		return nil
	}
	if c.pending == "" {
		c.bufferStarted = now
	}
	c.pending += delta
	return c.readyChunks(false, now)
}

func (c *StreamChunker) Ready(now time.Time) []string {
	return c.readyChunks(false, now)
}

func (c *StreamChunker) Flush(now time.Time) []string {
	return c.readyChunks(true, now)
}

func (c *StreamChunker) readyChunks(force bool, now time.Time) []string {
	var chunks []string
	for {
		chunk, rest, ok := c.selectChunk(force, now)
		if !ok {
			break
		}
		chunks = append(chunks, chunk)
		c.pending = strings.TrimSpace(rest)
		if c.pending == "" {
			c.bufferStarted = time.Time{}
		} else {
			c.bufferStarted = now
		}
		c.firstChunkSent = true
	}
	return chunks
}

func (c *StreamChunker) selectChunk(force bool, now time.Time) (string, string, bool) {
	text := strings.TrimSpace(c.pending)
	if text == "" {
		return "", "", false
	}

	if force {
		return forceChunk(text)
	}

	wait := now.Sub(c.bufferStarted)
	maxWait := normalChunkMaxWait
	if !c.firstChunkSent {
		maxWait = firstChunkMaxWait
	}

	if cut := lastBoundaryCut(text, true); cut > 0 && runeLen(text[:cut]) >= minChunkRunes {
		return trimSplit(text, cut)
	}

	if cut := lastBoundaryCut(text, false); cut > 0 && runeLen(text[:cut]) >= targetChunkRunes {
		return trimSplit(text, cut)
	}

	if wait >= maxWait {
		if cut := lastBoundaryCut(text, false); cut > 0 && runeLen(text[:cut]) >= minChunkRunes {
			return trimSplit(text, cut)
		}
		if runeLen(text) >= targetChunkRunes {
			return trimSplit(text, cutAtRune(text, targetChunkRunes))
		}
	}

	if runeLen(text) >= hardMaxChunkRunes {
		if cut := bestBoundaryBefore(text, softMaxChunkRunes); cut > 0 {
			return trimSplit(text, cut)
		}
		return trimSplit(text, cutAtRune(text, softMaxChunkRunes))
	}

	return "", text, false
}

func SplitText(text string) []string {
	chunker := NewStreamChunker()
	chunker.pending = strings.TrimSpace(text)
	return chunker.Flush(time.Now())
}

func SplitReadyText(text string) ([]string, string) {
	chunker := NewStreamChunker()
	chunker.pending = strings.TrimSpace(text)
	chunker.bufferStarted = time.Now().Add(-normalChunkMaxWait)
	chunks := chunker.Ready(time.Now())
	return chunks, chunker.pending
}

func forceChunk(text string) (string, string, bool) {
	if runeLen(text) <= forceMinTailRunes || runeLen(text) <= hardMaxChunkRunes {
		return text, "", true
	}
	if cut := bestBoundaryBefore(text, softMaxChunkRunes); cut > 0 {
		return trimSplit(text, cut)
	}
	return trimSplit(text, cutAtRune(text, softMaxChunkRunes))
}

func bestBoundaryBefore(text string, maxRunes int) int {
	limit := cutAtRune(text, maxRunes)
	best := 0
	for i, r := range text {
		end := i + utf8.RuneLen(r)
		if end > limit {
			break
		}
		if isBoundary(r) {
			best = end
		}
	}
	return best
}

func lastBoundaryCut(text string, strongOnly bool) int {
	cut := 0
	for i, r := range text {
		if strongOnly {
			if !isStrongBoundary(r) {
				continue
			}
		} else if !isSoftBoundary(r) {
			continue
		}
		cut = i + utf8.RuneLen(r)
	}
	return cut
}

func trimSplit(text string, cut int) (string, string, bool) {
	chunk := strings.TrimSpace(text[:cut])
	rest := strings.TrimSpace(text[cut:])
	if chunk == "" {
		return "", rest, false
	}
	return chunk, rest, true
}

func cutAtRune(text string, n int) int {
	if n <= 0 {
		return 0
	}
	count := 0
	for i, r := range text {
		count++
		if count == n {
			return i + utf8.RuneLen(r)
		}
	}
	return len(text)
}

func runeLen(text string) int {
	return utf8.RuneCountInString(text)
}

func isBoundary(r rune) bool {
	return isStrongBoundary(r) || isSoftBoundary(r)
}

func isStrongBoundary(r rune) bool {
	switch r {
	case 0x3002, 0xff01, 0xff1f, 0xff1b, '.', '!', '?', ';':
		return true
	default:
		return false
	}
}

func isSoftBoundary(r rune) bool {
	switch r {
	case 0xff0c, 0x3001, 0xff1a, ',', ':':
		return true
	default:
		return false
	}
}
