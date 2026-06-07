package tts

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Session struct {
	id  string
	cfg Config

	ctx    context.Context
	cancel context.CancelFunc
	textCh chan string
	textMu sync.RWMutex
	closed bool

	connMu sync.Mutex
	conn   *websocket.Conn
}

func NewSession(id string, cfg Config) *Session {
	ctx, cancel := context.WithCancel(context.Background())
	return &Session{
		id:     id,
		cfg:    cfg,
		ctx:    ctx,
		cancel: cancel,
		textCh: make(chan string, 64),
	}
}

func (s *Session) Start() error {
	if err := s.ctx.Err(); err != nil {
		return err
	}
	if err := s.connect(); err != nil {
		s.close()
		return err
	}
	if err := s.ctx.Err(); err != nil {
		s.close()
		return err
	}
	go s.readLoop()
	go s.run()
	return nil
}

func (s *Session) Stop() {
	s.cancel()
	s.textMu.Lock()
	if !s.closed {
		close(s.textCh)
		s.closed = true
	}
	s.textMu.Unlock()
	s.close()
}

func (s *Session) Enqueue(text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	s.textMu.RLock()
	defer s.textMu.RUnlock()
	if s.closed {
		return
	}
	if len(s.textCh) >= 4 {
		dropped := 0
		for {
			select {
			case <-s.textCh:
				dropped++
			default:
				if dropped > 0 {
					log.Printf("[tts] catch up by dropping stale queued text session=%s dropped=%d", s.id, dropped)
					if s.cfg.OnReset != nil {
						s.cfg.OnReset(s.id)
					}
				}
				goto enqueue
			}
		}
	}
enqueue:
	select {
	case s.textCh <- text:
	default:
		select {
		case dropped := <-s.textCh:
			log.Printf("[tts] dropping stale text because queue is full session=%s text=%q", s.id, dropped)
		default:
		}
		select {
		case s.textCh <- text:
		default:
			log.Printf("[tts] dropping text because queue is full session=%s text=%q", s.id, text)
		}
	}
}

func (s *Session) run() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case text, ok := <-s.textCh:
			if !ok {
				return
			}
			for _, chunk := range SplitText(text) {
				log.Printf("[tts] send text session=%s runes=%d text=%q", s.id, runeLen(chunk), chunk)
				if err := s.sendEvent(map[string]any{
					"type": "input_text_buffer.append",
					"text": chunk,
				}); err != nil {
					log.Printf("[tts] append failed session=%s err=%v", s.id, err)
					return
				}
				if err := s.sendEvent(map[string]any{
					"type": "input_text_buffer.commit",
				}); err != nil {
					log.Printf("[tts] commit failed session=%s err=%v", s.id, err)
					return
				}
			}
		}
	}
}

func (s *Session) connect() error {
	if err := s.ctx.Err(); err != nil {
		return err
	}

	u, err := url.Parse(s.cfg.RealtimeURL)
	if err != nil {
		return err
	}
	q := u.Query()
	if q.Get("model") == "" {
		q.Set("model", s.cfg.Model)
	}
	u.RawQuery = q.Encode()

	header := http.Header{}
	header.Set("Authorization", "Bearer "+s.cfg.DashScopeAPIKey)
	header.Set("X-DashScope-DataInspection", "enable")

	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	log.Printf("[tts] connecting dashscope session=%s url=%s", s.id, u.String())
	conn, _, err := dialer.DialContext(s.ctx, u.String(), header)
	if err != nil {
		return err
	}
	s.conn = conn
	log.Printf("[tts] dashscope connected session=%s", s.id)

	return s.sendEvent(map[string]any{
		"type": "session.update",
		"session": map[string]any{
			"mode":            "commit",
			"voice":           emptyDefault(s.cfg.Voice, "Cherry"),
			"language_type":   emptyDefault(s.cfg.Language, "Chinese"),
			"response_format": "pcm",
			"sample_rate":     24000,
		},
	})
}

func (s *Session) readLoop() {
	defer s.close()
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		_, raw, err := s.conn.ReadMessage()
		if err != nil {
			if s.ctx.Err() == nil {
				log.Printf("[tts] read failed session=%s err=%v", s.id, err)
			}
			return
		}

		var event map[string]any
		if err := json.Unmarshal(raw, &event); err != nil {
			log.Printf("[tts] parse event failed session=%s err=%v", s.id, err)
			continue
		}

		eventType, _ := event["type"].(string)
		switch eventType {
		case "response.audio.delta":
			delta, _ := event["delta"].(string)
			audio, err := base64.StdEncoding.DecodeString(delta)
			if err != nil {
				log.Printf("[tts] decode audio failed session=%s err=%v", s.id, err)
				continue
			}
			log.Printf("[tts] audio delta session=%s bytes=%d", s.id, len(audio))
			if s.cfg.OnAudio != nil {
				s.cfg.OnAudio(s.id, audio)
			}
		case "error":
			log.Printf("[tts] dashscope error session=%s payload=%s", s.id, string(raw))
		case "session.created", "session.updated", "input_text_buffer.committed", "response.created", "response.audio.done", "response.done", "session.finished":
			log.Printf("[tts] event session=%s type=%s", s.id, eventType)
		}
	}
}

func (s *Session) sendEvent(event map[string]any) error {
	event["event_id"] = fmt.Sprintf("event_%d", time.Now().UnixNano())
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	s.connMu.Lock()
	defer s.connMu.Unlock()
	if s.conn == nil {
		return fmt.Errorf("tts connection is closed")
	}
	return s.conn.WriteMessage(websocket.TextMessage, payload)
}

func (s *Session) close() {
	s.connMu.Lock()
	if s.conn != nil {
		_ = s.conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"session.finish"}`))
		_ = s.conn.Close()
		s.conn = nil
	}
	s.connMu.Unlock()

}

func emptyDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
