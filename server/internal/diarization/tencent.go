package diarization

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const tencentRealtimeHost = "asr.cloud.tencent.com"

type tencentStream struct {
	cfg       Config
	sessionID string
	conn      *websocket.Conn
	mu        sync.Mutex
	closed    bool
	buffer    []byte
	timer     *time.Timer
}

type tencentResponse struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	VoiceID   string `json:"voice_id"`
	MessageID string `json:"message_id"`
	Result    struct {
		VoiceTextStr string `json:"voice_text_str"`
		Text         string `json:"text"`
		SliceType    int    `json:"slice_type"`
		SpeakerID    int    `json:"speaker_id"`
		Sentences    any    `json:"sentences"`
	} `json:"result"`
	Sentences struct {
		SentenceList []struct {
			Sentence     string `json:"sentence"`
			SentenceType int    `json:"sentence_type"`
			SpeakerID    int    `json:"speaker_id"`
		} `json:"sentence_list"`
	} `json:"sentences"`
}

func newTencentStream(cfg Config, sessionID string) (*tencentStream, error) {
	endpoint, err := tencentURL(cfg, sessionID, time.Now().Unix())
	if err != nil {
		return nil, err
	}

	conn, _, err := websocket.DefaultDialer.Dial(endpoint, http.Header{})
	if err != nil {
		return nil, err
	}

	s := &tencentStream{
		cfg:       cfg,
		sessionID: sessionID,
		conn:      conn,
	}
	go s.readLoop()
	log.Printf("[diarization] tencent stream started session=%s", sessionID)
	return s, nil
}

func tencentURL(cfg Config, voiceID string, timestamp int64) (string, error) {
	if cfg.AppID == "" || cfg.SecretID == "" || cfg.SecretKey == "" {
		return "", fmt.Errorf("missing Tencent ASR credentials")
	}
	model := cfg.Model
	if model == "" {
		model = "16k_zh_en_speaker"
	}

	params := map[string]string{
		"secretid":          cfg.SecretID,
		"timestamp":         strconv.FormatInt(timestamp, 10),
		"expired":           strconv.FormatInt(timestamp+24*60*60, 10),
		"nonce":             strconv.FormatInt(timestamp%1000000000, 10),
		"engine_model_type": model,
		"voice_format":      "1",
		"voice_id":          voiceID,
	}

	signature := tencentSignature(tencentRealtimeHost, "/asr/v2/"+cfg.AppID, params, cfg.SecretKey)
	q := url.Values{}
	for key, value := range params {
		q.Set(key, value)
	}
	q.Set("signature", signature)
	return "wss://" + tencentRealtimeHost + "/asr/v2/" + cfg.AppID + "?" + q.Encode(), nil
}

func tencentSignature(host, path string, params map[string]string, secretKey string) string {
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	pairs := make([]string, 0, len(keys))
	for _, key := range keys {
		pairs = append(pairs, key+"="+params[key])
	}
	plainText := host + path + "?" + strings.Join(pairs, "&")
	mac := hmac.New(sha1.New, []byte(secretKey))
	mac.Write([]byte(plainText))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func (s *tencentStream) SendPCM(pcm []byte) error {
	pcm16k := downsample48kTo16kPCM16(pcm)
	if len(pcm16k) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.buffer = append(s.buffer, pcm...)
	s.resetFallbackTimerLocked()
	return s.conn.WriteMessage(websocket.BinaryMessage, pcm16k)
}

func (s *tencentStream) Close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	if s.timer != nil {
		s.timer.Stop()
	}
	_ = s.conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"end"}`))
	_ = s.conn.Close()
	s.mu.Unlock()
	log.Printf("[diarization] tencent stream stopped session=%s", s.sessionID)
}

func (s *tencentStream) resetFallbackTimerLocked() {
	if s.cfg.OnFallback == nil {
		return
	}
	if s.timer != nil {
		s.timer.Stop()
	}
	s.timer = time.AfterFunc(2*time.Second, func() {
		s.mu.Lock()
		if s.closed || len(s.buffer) == 0 {
			s.mu.Unlock()
			return
		}
		pcm := make([]byte, len(s.buffer))
		copy(pcm, s.buffer)
		s.buffer = nil
		s.mu.Unlock()
		s.cfg.OnFallback(s.sessionID, AudioFallback{PCM: pcm})
	})
}

func (s *tencentStream) readLoop() {
	for {
		_, msg, err := s.conn.ReadMessage()
		if err != nil {
			s.mu.Lock()
			closed := s.closed
			s.mu.Unlock()
			if !closed {
				log.Printf("[diarization] tencent read failed session=%s err=%v", s.sessionID, err)
			}
			return
		}

		var resp tencentResponse
		if err := json.Unmarshal(msg, &resp); err != nil {
			continue
		}
		log.Printf("[diarization] tencent raw session=%s body=%s", s.sessionID, string(msg))
		if resp.Code != 0 {
			log.Printf("[diarization] tencent response error session=%s code=%d message=%s", s.sessionID, resp.Code, resp.Message)
			continue
		}
		text := strings.TrimSpace(resp.Result.VoiceTextStr)
		if text == "" {
			text = strings.TrimSpace(resp.Result.Text)
		}
		if text == "" {
			text = finalTencentSentenceText(resp.Sentences.SentenceList)
		}
		if text == "" {
			continue
		}
		if resp.Result.SliceType != 2 && !hasFinalTencentSentence(resp.Sentences.SentenceList) {
			continue
		}
		speakerID := resp.Result.SpeakerID
		if speakerID <= 0 {
			speakerID = finalTencentSentenceSpeaker(resp.Sentences.SentenceList)
		}
		speaker := tencentSpeakerLabel(speakerID, resp.Result.Sentences)
		s.mu.Lock()
		s.buffer = nil
		if s.timer != nil {
			s.timer.Stop()
		}
		s.mu.Unlock()
		if s.cfg.OnFinal != nil {
			s.cfg.OnFinal(s.sessionID, Transcript{Text: text, Speaker: speaker})
		}
	}
}

func finalTencentSentenceText(sentences []struct {
	Sentence     string `json:"sentence"`
	SentenceType int    `json:"sentence_type"`
	SpeakerID    int    `json:"speaker_id"`
}) string {
	for i := len(sentences) - 1; i >= 0; i-- {
		if sentences[i].SentenceType == 1 {
			return strings.TrimSpace(sentences[i].Sentence)
		}
	}
	return ""
}

func hasFinalTencentSentence(sentences []struct {
	Sentence     string `json:"sentence"`
	SentenceType int    `json:"sentence_type"`
	SpeakerID    int    `json:"speaker_id"`
}) bool {
	return finalTencentSentenceText(sentences) != ""
}

func finalTencentSentenceSpeaker(sentences []struct {
	Sentence     string `json:"sentence"`
	SentenceType int    `json:"sentence_type"`
	SpeakerID    int    `json:"speaker_id"`
}) int {
	for i := len(sentences) - 1; i >= 0; i-- {
		if sentences[i].SentenceType == 1 {
			return sentences[i].SpeakerID
		}
	}
	return 0
}

func tencentSpeakerLabel(speakerID int, sentences any) string {
	if speakerID < 0 {
		speakerID = speakerFromSentences(sentences)
	}
	if speakerID < 0 {
		return ""
	}
	idx := speakerID
	if idx >= 0 && idx < 26 {
		return string(rune('A' + idx))
	}
	return "S" + strconv.Itoa(speakerID)
}

func speakerFromSentences(sentences any) int {
	switch value := sentences.(type) {
	case []any:
		counts := map[int]int{}
		best, bestCount := 0, 0
		for _, item := range value {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			id := numberField(m, "speaker_id")
			if id <= 0 {
				id = numberField(m, "speaker")
			}
			if id <= 0 {
				continue
			}
			counts[id]++
			if counts[id] > bestCount {
				best, bestCount = id, counts[id]
			}
		}
		return best
	case map[string]any:
		id := numberField(value, "speaker_id")
		if id <= 0 {
			id = numberField(value, "speaker")
		}
		return id
	default:
		return 0
	}
}

func numberField(values map[string]any, key string) int {
	switch value := values[key].(type) {
	case float64:
		return int(value)
	case int:
		return value
	case string:
		n, _ := strconv.Atoi(value)
		return n
	default:
		return 0
	}
}

func downsample48kTo16kPCM16(pcm []byte) []byte {
	if len(pcm) < 6 {
		return nil
	}
	sampleCount := len(pcm) / 2
	outSamples := sampleCount / 3
	out := make([]byte, outSamples*2)
	for i := 0; i < outSamples; i++ {
		src := i * 3 * 2
		copy(out[i*2:i*2+2], pcm[src:src+2])
	}
	return out
}

func pcm16Sample(pcm []byte, index int) int16 {
	return int16(binary.LittleEndian.Uint16(pcm[index*2 : index*2+2]))
}
