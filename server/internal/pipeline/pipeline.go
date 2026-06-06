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
)

// Pipeline orchestrates the ASR → translate → correct → SSE pipeline.
type Pipeline struct {
	cfg     *config.Config
	mgr     *session.Manager
	broker  *sse.Broker
	asr     *ASRClient
	transl  *Translator
	corr    *Corrector
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
		corr:       NewCorrector(),
		segmenters: make(map[string]*audio.Segmenter),
	}
}

// HandleCreateSession — POST /api/v1/sessions
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

// HandleUploadAudio — POST /api/v1/sessions/{sessionId}/audio
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
		// Silent or unrecognized audio — skip
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"status":"ok"}`))
		return
	}

	recentContext := sess.GetWindow(5)
	chineseText, err := p.transl.TranslateWithContext(englishText, recentContext)
	if err != nil {
		log.Printf("[ERROR] translate failed session=%s err=%v", sessionID, err)
		chineseText = "[翻译失败]"
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

// HandleStopSession — POST /api/v1/sessions/{sessionId}/stop
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
