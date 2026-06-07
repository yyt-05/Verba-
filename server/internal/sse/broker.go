package sse

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
)

// Event types as defined in the PRD review.
const (
	EventSessionStatus     = "session.status"
	EventSubtitlePartial   = "subtitle.partial"
	EventSubtitleFinal     = "subtitle.final"
	EventSubtitleCorrected = "subtitle.corrected"
	EventTTSAudioDelta     = "tts.audio.delta"
	EventTTSAudioReset     = "tts.audio.reset"
	EventBackgroundSummary = "background.summary"
	EventWarning           = "warning"
	EventError             = "error"
)

// Event represents a single SSE event pushed to the client.
type Event struct {
	ID         int             `json:"id"`
	Type       string          `json:"type"`
	Data       json.RawMessage `json:"data"`
	Timestamp  int64           `json:"timestamp"`
	EventSeq   int             `json:"eventSeq,omitempty"`
	SegmentID  int             `json:"segmentId,omitempty"`
	SegmentSeq int             `json:"segmentSeq,omitempty"`
	Revision   int             `json:"revision,omitempty"`
	Status     string          `json:"status,omitempty"`
	IsFinal    bool            `json:"isFinal,omitempty"`
	OldText    string          `json:"oldText,omitempty"`
	NewText    string          `json:"newText,omitempty"`
}

// Broker manages SSE connections per session.
type Broker struct {
	subscribers map[string][]chan Event
	mu          sync.RWMutex
}

func NewBroker() *Broker {
	return &Broker{
		subscribers: make(map[string][]chan Event),
	}
}

// Subscribe creates a new channel for a session.
func (b *Broker) Subscribe(sessionID string) chan Event {
	ch := make(chan Event, 256)
	b.mu.Lock()
	b.subscribers[sessionID] = append(b.subscribers[sessionID], ch)
	b.mu.Unlock()
	return ch
}

// Unsubscribe removes a channel from a session.
func (b *Broker) Unsubscribe(sessionID string, ch chan Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	subs := b.subscribers[sessionID]
	for i, sub := range subs {
		if sub == ch {
			b.subscribers[sessionID] = append(subs[:i], subs[i+1:]...)
			close(ch)
			return
		}
	}
}

// Publish sends an event to all subscribers of a session.
func (b *Broker) Publish(sessionID string, evt Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.subscribers[sessionID] {
		select {
		case ch <- evt:
		default:
			// drop if client is too slow
		}
	}
}

// HandleSSE is the HTTP handler for GET /api/v1/sessions/{sessionId}/events.
func (b *Broker) HandleSSE(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("sessionId")
	if sessionID == "" {
		http.Error(w, "missing sessionId", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	ch := b.Subscribe(sessionID)
	defer b.Unsubscribe(sessionID, ch)

	// Send initial connected status
	fmt.Fprintf(w, "event: session.status\ndata: {\"status\":\"connected\"}\n\n")
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-ch:
			if !ok {
				return
			}
			payload, _ := json.Marshal(evt)
			fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", evt.ID, evt.Type, payload)
			flusher.Flush()
		}
	}
}

// BuildSubtitleFinal creates a finalized subtitle event.
func BuildSubtitleFinal(id int, segmentID int, original, translation, speaker string) Event {
	body, _ := json.Marshal(map[string]interface{}{
		"segmentId":   segmentID,
		"segmentSeq":  segmentID,
		"original":    original,
		"translation": translation,
		"speaker":     speaker,
		"revision":    1,
		"status":      "final",
		"isFinal":     true,
	})
	return Event{
		ID:         id,
		Type:       EventSubtitleFinal,
		Data:       body,
		EventSeq:   id,
		SegmentID:  segmentID,
		SegmentSeq: segmentID,
		Revision:   1,
		Status:     "final",
		IsFinal:    true,
	}
}

// BuildCorrection creates a correction event.
func BuildCorrection(id int, segmentID int, oldText, newText string, revision int) Event {
	body, _ := json.Marshal(map[string]interface{}{
		"segmentId":  segmentID,
		"segmentSeq": segmentID,
		"oldText":    oldText,
		"newText":    newText,
		"revision":   revision,
		"status":     "corrected",
		"isFinal":    true,
	})
	return Event{
		ID:         id,
		Type:       EventSubtitleCorrected,
		Data:       body,
		EventSeq:   id,
		SegmentID:  segmentID,
		SegmentSeq: segmentID,
		OldText:    oldText,
		NewText:    newText,
		Revision:   revision,
		Status:     "corrected",
		IsFinal:    true,
	}
}

// BuildTTSAudioDelta creates a TTS PCM audio chunk event.
func BuildTTSAudioDelta(id int, audio []byte) Event {
	body, _ := json.Marshal(map[string]interface{}{
		"audio":      base64.StdEncoding.EncodeToString(audio),
		"encoding":   "pcm_s16le",
		"sampleRate": 24000,
		"channels":   1,
	})
	return Event{
		ID:       id,
		Type:     EventTTSAudioDelta,
		Data:     body,
		EventSeq: id,
	}
}

// BuildTTSAudioReset tells the client to drop queued TTS audio and catch up.
func BuildTTSAudioReset(id int) Event {
	body, _ := json.Marshal(map[string]interface{}{
		"reason": "catch_up",
	})
	return Event{
		ID:       id,
		Type:     EventTTSAudioReset,
		Data:     body,
		EventSeq: id,
	}
}

// BuildBackgroundSummary creates an event carrying the AI-generated context summary.
func BuildBackgroundSummary(id int, summary string, sentenceCount int) Event {
	body, _ := json.Marshal(map[string]interface{}{
		"summary":       summary,
		"sentenceCount": sentenceCount,
	})
	return Event{
		ID:       id,
		Type:     EventBackgroundSummary,
		Data:     body,
		EventSeq: id,
	}
}

var _ = log.Println
