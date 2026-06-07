package diarization

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

type deepgramStream struct {
	cfg       Config
	sessionID string
	conn      *websocket.Conn
	mu        sync.Mutex
	closed    bool
}

type deepgramResponse struct {
	Type      string `json:"type"`
	IsFinal   bool   `json:"is_final"`
	SpeechEnd bool   `json:"speech_final"`
	Channel   struct {
		Alternatives []struct {
			Transcript string `json:"transcript"`
			Words      []struct {
				Speaker int `json:"speaker"`
			} `json:"words"`
		} `json:"alternatives"`
	} `json:"channel"`
}

func newDeepgramStream(cfg Config, sessionID string) (*deepgramStream, error) {
	endpoint, err := deepgramURL(cfg)
	if err != nil {
		return nil, err
	}

	header := http.Header{}
	header.Set("Authorization", "Token "+cfg.APIKey)
	conn, _, err := websocket.DefaultDialer.Dial(endpoint, header)
	if err != nil {
		return nil, err
	}

	s := &deepgramStream{
		cfg:       cfg,
		sessionID: sessionID,
		conn:      conn,
	}
	go s.readLoop()
	log.Printf("[diarization] deepgram stream started session=%s", sessionID)
	return s, nil
}

func deepgramURL(cfg Config) (string, error) {
	u, err := url.Parse(cfg.RealtimeURL)
	if err != nil {
		return "", err
	}
	q := u.Query()
	setDefaultQuery(q, "model", cfg.Model)
	setDefaultQuery(q, "language", cfg.Language)
	setDefaultQuery(q, "encoding", "linear16")
	setDefaultQuery(q, "sample_rate", "48000")
	setDefaultQuery(q, "channels", "1")
	setDefaultQuery(q, "diarize", "true")
	setDefaultQuery(q, "interim_results", "false")
	setDefaultQuery(q, "smart_format", "false")
	setDefaultQuery(q, "punctuate", "true")
	setDefaultQuery(q, "endpointing", "500")
	setDefaultQuery(q, "vad_events", "false")
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func setDefaultQuery(q url.Values, key, value string) {
	if q.Get(key) == "" && value != "" {
		q.Set(key, value)
	}
}

func (s *deepgramStream) SendPCM(pcm []byte) error {
	if len(pcm) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	return s.conn.WriteMessage(websocket.BinaryMessage, pcm)
}

func (s *deepgramStream) Close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	_ = s.conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"CloseStream"}`))
	_ = s.conn.Close()
	s.mu.Unlock()
	log.Printf("[diarization] deepgram stream stopped session=%s", s.sessionID)
}

func (s *deepgramStream) readLoop() {
	for {
		_, msg, err := s.conn.ReadMessage()
		if err != nil {
			s.mu.Lock()
			closed := s.closed
			s.mu.Unlock()
			if !closed {
				log.Printf("[diarization] deepgram read failed session=%s err=%v", s.sessionID, err)
			}
			return
		}

		var resp deepgramResponse
		if err := json.Unmarshal(msg, &resp); err != nil {
			continue
		}
		if !resp.IsFinal && !resp.SpeechEnd {
			continue
		}
		if len(resp.Channel.Alternatives) == 0 {
			continue
		}
		alt := resp.Channel.Alternatives[0]
		text := strings.TrimSpace(alt.Transcript)
		if text == "" {
			continue
		}

		speaker := speakerLabelFromWords(alt.Words)
		if s.cfg.OnFinal != nil {
			s.cfg.OnFinal(s.sessionID, Transcript{
				Text:    text,
				Speaker: speaker,
			})
		}
	}
}

func speakerLabelFromWords(words []struct {
	Speaker int `json:"speaker"`
}) string {
	if len(words) == 0 {
		return ""
	}
	counts := map[int]int{}
	bestSpeaker := words[0].Speaker
	bestCount := 0
	for _, word := range words {
		counts[word.Speaker]++
		if counts[word.Speaker] > bestCount {
			bestSpeaker = word.Speaker
			bestCount = counts[word.Speaker]
		}
	}
	if bestSpeaker >= 0 && bestSpeaker < 26 {
		return string(rune('A' + bestSpeaker))
	}
	return "S" + strconv.Itoa(bestSpeaker)
}
