package pipeline

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/verba/server/internal/audio"
	"github.com/verba/server/internal/config"
	"github.com/verba/server/internal/session"
	"github.com/verba/server/internal/sse"
	"github.com/verba/server/internal/tts"
)

// Pipeline orchestrates the ASR 鈫?translate 鈫?correct 鈫?SSE pipeline.
type Pipeline struct {
	cfg     *config.Config
	mgr     *session.Manager
	broker  *sse.Broker
	asr     *ASRClient
	transl  *Translator
	corr    *Corrector
	tts     *tts.Manager
	eventID int

	segmenters   map[string]*audio.Segmenter
	segmentersMu sync.Mutex
}

func New(cfg *config.Config, mgr *session.Manager, broker *sse.Broker) *Pipeline {
	return &Pipeline{
		cfg:    cfg,
		mgr:    mgr,
		broker: broker,
		asr: NewASRClient(ASRConfig{
			APIKey:  cfg.SiliconFlowAPIKey,
			BaseURL: cfg.SiliconFlowBaseURL,
			Model:   cfg.ASRModel,
		}),
		transl: NewTranslator(TranslatorConfig{
			APIKey:  cfg.SiliconFlowAPIKey,
			BaseURL: cfg.SiliconFlowBaseURL,
			Model:   cfg.TranslateModel,
		}),
		tts: tts.NewManager(tts.Config{
			Provider:        cfg.TTSProvider,
			DashScopeAPIKey: cfg.DashScopeAPIKey,
			RealtimeURL:     cfg.DashScopeRealtimeURL,
			Model:           cfg.TTSModel,
			Voice:           cfg.TTSVoice,
			Language:        cfg.TTSLanguage,
		}),
		corr:       NewCorrector(),
		segmenters: make(map[string]*audio.Segmenter),
	}
}

// HandleCreateSession 鈥?POST /api/v1/sessions
func (p *Pipeline) HandleCreateSession(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	id := fmt.Sprintf("sess_%d", time.Now().UnixMilli())
	sess := p.mgr.Create(id)
	p.segmenterFor(id)

	log.Printf("[http] POST /api/v1/sessions session=%s status=%s dur=%dms",
		id, sess.Status, time.Since(start).Milliseconds())

	statusData, _ := json.Marshal(map[string]string{"status": string(sess.Status), "sessionId": id})
	p.broker.Publish(id, sse.Event{
		ID:   0,
		Type: sse.EventSessionStatus,
		Data: statusData,
	})

	resp, _ := json.Marshal(map[string]string{"session_id": id})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(resp)
}

// HandleUploadAudio 鈥?POST /api/v1/sessions/{sessionId}/audio
func (p *Pipeline) HandleUploadAudio(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	sessionID := r.PathValue("sessionId")
	sess := p.mgr.Get(sessionID)
	if sess == nil {
		log.Printf("[WARN] audio upload session not found session=%s", sessionID)
		http.Error(w, `{"error":"session not found"}`, http.StatusNotFound)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil || len(body) == 0 {
		log.Printf("[WARN] audio upload empty body session=%s", sessionID)
		http.Error(w, `{"error":"empty body"}`, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	rms := audio.PCMRMS(body)
	log.Printf("[audio] received session=%s bytes=%d rms=%.1f dur=%dms",
		sessionID, len(body), rms, time.Since(start).Milliseconds())

	segment, ready := p.segmenterFor(sessionID).AddPCM(body)
	if !ready {
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"status":"buffering"}`))
		return
	}

	log.Printf("[audio] segment ready session=%s bytes=%d", sessionID, len(segment))

	wavAudio := audio.WriteWAVHeader(48000, segment)
	englishText, err := p.asr.Transcribe(wavAudio)
	if err != nil {
		log.Printf("[ERROR] asr failed session=%s err=%v", sessionID, err)
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"status":"ok"}`))
		return
	}
	if englishText == "" {
		// Silent or unrecognized audio 鈥?skip
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"status":"ok"}`))
		return
	}

	recentContext := sess.GetWindow(5)
	chineseText, err := p.translateForSession(sessionID, englishText, recentContext)
	if err != nil {
		log.Printf("[ERROR] translate failed session=%s err=%v", sessionID, err)
		chineseText = "[缈昏瘧澶辫触]"
	}

	p.eventID++
	seg, shouldCorrect := sess.AppendSentence(englishText, chineseText)

	evt := sse.BuildSubtitleFinal(p.eventID, seg.Index, seg.Original, seg.Translation)
	p.broker.Publish(sessionID, evt)
	log.Printf("[sse] published subtitle.final session=%s segmentId=%d", sessionID, seg.Index)

	// Correction engine
	if shouldCorrect {
		p.eventID++
		window := sess.GetWindow(p.corr.WindowSize)
		lookbackStart := len(window) - p.corr.LookbackCount
		if lookbackStart < 0 {
			lookbackStart = 0
		}
		lookback := window[lookbackStart:]
		log.Printf("[corrector] triggered session=%s window=%d lookback=%d",
			sessionID, len(window), len(lookback))

		prompt := p.corr.BuildCorrectionPrompt(window, lookback)
		suggestions, err := p.transl.Correct(prompt)
		if err != nil {
			log.Printf("[ERROR] corrector call failed session=%s err=%v", sessionID, err)
		} else {
			for _, suggestion := range suggestions {
				if suggestion.Confidence < 0.6 {
					continue
				}
				newRev := 1
				for _, w := range window {
					if w.Index == suggestion.SegmentIndex {
						newRev = w.Revision + 1
						break
					}
				}
				oldText := ""
				for _, w := range window {
					if w.Index == suggestion.SegmentIndex {
						oldText = w.Translation
						break
					}
				}
				if sess.ApplyCorrection(suggestion.SegmentIndex, suggestion.NewTranslation, newRev) {
					corrEvt := sse.BuildCorrection(p.eventID, suggestion.SegmentIndex,
						oldText, suggestion.NewTranslation, newRev)
					p.broker.Publish(sessionID, corrEvt)
					log.Printf("[sse] published subtitle.corrected session=%s segmentId=%d rev=%d",
						sessionID, suggestion.SegmentIndex, newRev)
				}
			}
		}
	}

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(`{"status":"ok"}`))
}

// HandleStopSession 鈥?POST /api/v1/sessions/{sessionId}/stop
func (p *Pipeline) translateForSession(sessionID, englishText string, recent []session.Sentence) (string, error) {
	if !p.tts.Enabled(sessionID) {
		return p.transl.TranslateWithContext(englishText, recent)
	}

	var pending string
	flush := func(force bool) {
		if pending == "" {
			return
		}
		chunks := tts.SplitText(pending)
		if !force && len(chunks) == 1 && chunks[0] == pending {
			return
		}
		pending = ""
		for i, chunk := range chunks {
			if !force && i == len(chunks)-1 && !endsWithBoundary(chunk) {
				pending = chunk
				continue
			}
			p.tts.Speak(sessionID, chunk)
		}
	}

	result, err := p.transl.TranslateStreamWithContext(englishText, recent, func(delta string) {
		pending += delta
		flush(false)
	})
	if err != nil {
		return "", err
	}
	flush(true)
	return result, nil
}

func endsWithBoundary(text string) bool {
	runes := []rune(text)
	if len(runes) == 0 {
		return false
	}
	last := runes[len(runes)-1]
	switch last {
	case 0xff0c, 0x3002, 0xff01, 0xff1f, 0xff1b, 0xff1a, 0x3001, ',', '.', '!', '?', ';', ':':
		return true
	default:
		return false
	}
}

// HandleTTSControl toggles realtime Chinese speech for a session.
func (p *Pipeline) HandleTTSControl(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("sessionId")
	if p.mgr.Get(sessionID) == nil {
		http.Error(w, `{"error":"session not found"}`, http.StatusNotFound)
		return
	}

	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}

	if body.Enabled {
		if err := p.tts.Enable(sessionID); err != nil {
			log.Printf("[tts] enable failed session=%s err=%v", sessionID, err)
			http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusBadRequest)
			return
		}
	} else {
		p.tts.Disable(sessionID)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

func (p *Pipeline) HandleStopSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("sessionId")
	sess := p.mgr.Get(sessionID)
	if sess == nil {
		log.Printf("[WARN] stop session not found session=%s", sessionID)
		http.Error(w, `{"error":"session not found"}`, http.StatusNotFound)
		return
	}

	duration := time.Since(sess.CreatedAt)
	sess.SetStatus(session.StatusStopped)
	p.tts.Disable(sessionID)

	statusData, _ := json.Marshal(map[string]string{"status": "stopped"})
	p.broker.Publish(sessionID, sse.Event{
		Type: sse.EventSessionStatus,
		Data: statusData,
	})

	go func() {
		time.Sleep(5 * time.Second)
		p.mgr.Remove(sessionID)
		p.removeSegmenter(sessionID)
	}()

	log.Printf("[session] stopped id=%s dur=%s", sessionID, duration.Round(time.Second))
	w.Write([]byte(`{"status":"stopped"}`))
}

var _ = io.Discard
var _ = json.Marshal

func (p *Pipeline) segmenterFor(sessionID string) *audio.Segmenter {
	p.segmentersMu.Lock()
	defer p.segmentersMu.Unlock()

	seg := p.segmenters[sessionID]
	if seg == nil {
		seg = audio.NewSegmenter(48000)
		p.segmenters[sessionID] = seg
	}
	return seg
}

func (p *Pipeline) removeSegmenter(sessionID string) {
	p.segmentersMu.Lock()
	delete(p.segmenters, sessionID)
	p.segmentersMu.Unlock()
}
