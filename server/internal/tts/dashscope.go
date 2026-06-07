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
	id     string
	cfg    Config
	player *PCMPlayer

	ctx    context.Context
	cancel context.CancelFunc
	textCh chan string

	connMu sync.Mutex
	conn   *websocket.Conn
	audio  *AudioStream
}

func NewSession(id string, cfg Config, player *PCMPlayer) *Session {
	ctx, cancel := context.WithCancel(context.Background())
	return &Session{
		id:     id,
		cfg:    cfg,
		player: player,
		ctx:    ctx,
		cancel: cancel,
		textCh: make(chan string, 64),
	}
}

func (s *Session) Start() {
	go s.run()
}

func (s *Session) Stop() {
	s.cancel()
	close(s.textCh)
	s.close()
}

func (s *Session) Enqueue(text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	select {
	case s.textCh <- text:
	default:
		log.Printf("[tts] dropping text because queue is full session=%s", s.id)
	}
}

func (s *Session) run() {
	if err := s.connect(); err != nil {
		log.Printf("[tts] connect failed session=%s err=%v", s.id, err)
		return
	}

	go s.readLoop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case text, ok := <-s.textCh:
			if !ok {
				return
			}
			for _, chunk := range SplitText(text) {
				if err := s.sendEvent(map[string]any{
					"type": "input_text_buffer.append",
					"text": chunk,
				}); err != nil {
					log.Printf("[tts] append failed session=%s err=%v", s.id, err)
					return
				}
				time.Sleep(80 * time.Millisecond)
			}
		}
	}
}

func (s *Session) connect() error {
	audio, err := s.player.NewStream()
	if err != nil {
		return fmt.Errorf("audio stream: %w", err)
	}
	s.audio = audio

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

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), header)
	if err != nil {
		audio.Close()
		return err
	}
	s.conn = conn

	return s.sendEvent(map[string]any{
		"type": "session.update",
		"session": map[string]any{
			"mode":            "server_commit",
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
			if err := s.audio.Write(audio); err != nil && s.ctx.Err() == nil {
				log.Printf("[tts] audio write failed session=%s err=%v", s.id, err)
			}
		case "error":
			log.Printf("[tts] dashscope error session=%s payload=%s", s.id, string(raw))
		case "session.created", "session.updated", "response.created", "response.done", "session.finished":
			log.Printf("[tts] event session=%s type=%s", s.id, eventType)
		}
	}
}

func (s *Session) sendEvent(event map[string]any) error {
	event["event_id"] = fmt.Sprintf("event_%d", time.Now().UnixMilli())
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

	if s.audio != nil {
		s.audio.Close()
		s.audio = nil
	}
}

func emptyDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
