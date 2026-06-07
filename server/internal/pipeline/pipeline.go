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
	"github.com/verba/server/internal/diarization"
	"github.com/verba/server/internal/session"
	"github.com/verba/server/internal/sse"
	"github.com/verba/server/internal/tts"
)

// Pipeline orchestrates the ASR, translation, correction, and SSE pipeline.
type Pipeline struct {
	cfg     *config.Config
	mgr     *session.Manager
	broker  *sse.Broker
	asr     *ASRClient
	transl  *Translator
	corr    *Corrector
	diar    *diarization.Manager
	tts     *tts.Manager
	eventID int
	eventMu sync.Mutex

	segmenters    map[string]*audio.Segmenter
	segmentersMu  sync.Mutex
	ttsChunkers   map[string]*tts.StreamChunker
	ttsChunkersMu sync.Mutex
}

func New(cfg *config.Config, mgr *session.Manager, broker *sse.Broker) *Pipeline {
	diarizationModel := cfg.DeepgramModel
	if cfg.DiarizationProvider == "tencent" {
		diarizationModel = cfg.TencentASRModel
	}
	pipe := &Pipeline{
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
		corr:        NewCorrector(),
		segmenters:  make(map[string]*audio.Segmenter),
		ttsChunkers: make(map[string]*tts.StreamChunker),
	}
	pipe.diar = diarization.NewManager(diarization.Config{
		Provider:    cfg.DiarizationProvider,
		APIKey:      cfg.DeepgramAPIKey,
		RealtimeURL: cfg.DeepgramRealtimeURL,
		Model:       diarizationModel,
		Language:    cfg.DeepgramLanguage,
		AppID:       cfg.TencentASRAppID,
		SecretID:    cfg.TencentASRSecretID,
		SecretKey:   cfg.TencentASRSecretKey,
		OnFinal: func(sessionID string, transcript diarization.Transcript) {
			pipe.applyDiarizationSpeaker(sessionID, transcript.Speaker)
		},
		OnFallback: func(sessionID string, fallback diarization.AudioFallback) {
			// Main ASR path handles transcription; fallback does nothing
		},
	})
	pipe.tts = tts.NewManager(tts.Config{
		Provider:        cfg.TTSProvider,
		DashScopeAPIKey: cfg.DashScopeAPIKey,
		RealtimeURL:     cfg.DashScopeRealtimeURL,
		Model:           cfg.TTSModel,
		Voice:           cfg.TTSVoice,
		Language:        cfg.TTSLanguage,
		OnAudio: func(sessionID string, audio []byte) {
			pipe.publishTTSAudio(sessionID, audio)
		},
		OnReset: func(sessionID string) {
			pipe.publishTTSAudioReset(sessionID)
		},
	})
	return pipe
}

func (p *Pipeline) nextEventID() int {
	p.eventMu.Lock()
	defer p.eventMu.Unlock()
	p.eventID++
	return p.eventID
}

func (p *Pipeline) publishTTSAudio(sessionID string, audio []byte) {
	if len(audio) == 0 {
		return
	}
	p.eventMu.Lock()
	p.eventID++
	id := p.eventID
	p.eventMu.Unlock()
	p.broker.Publish(sessionID, sse.BuildTTSAudioDelta(id, audio))
}

func (p *Pipeline) publishTTSAudioReset(sessionID string) {
	p.eventMu.Lock()
	p.eventID++
	id := p.eventID
	p.eventMu.Unlock()
	p.broker.Publish(sessionID, sse.BuildTTSAudioReset(id))
}

// HandleCreateSession handles POST /api/v1/sessions.
func (p *Pipeline) HandleCreateSession(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	id := fmt.Sprintf("sess_%d", time.Now().UnixMilli())
	sess := p.mgr.Create(id)
	p.segmenterFor(id)
	if err := p.diar.Start(id); err != nil {
		log.Printf("[WARN] diarization start failed session=%s err=%v", id, err)
	}

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

// HandleUploadAudio handles POST /api/v1/sessions/{sessionId}/audio.
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

	// Send audio to diarization in parallel (speaker labels only, non-blocking).
	if p.diar.Enabled() {
		go func() {
			if err := p.diar.AddPCM(sessionID, body); err != nil {
				log.Printf("[WARN] diarization audio send failed session=%s err=%v", sessionID, err)
			}
		}()
	}

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
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"status":"ok"}`))
		return
	}
	if isChineseText(englishText) {
		// TTS echo — discard captured Chinese speech.
		log.Printf("[audio] discarded TTS echo session=%s text=%q", sessionID, englishText)
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"status":"ok"}`))
		return
		return
	}
	speaker := sess.GetCurrentSpeaker()
	if speaker == "" {
		speaker = "A"
	}
	p.handleFinalTranscript(sessionID, englishText, speaker)

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(`{"status":"ok"}`))
}

func (p *Pipeline) handleFinalTranscript(sessionID, englishText, speaker string) {
	sess := p.mgr.Get(sessionID)
	if sess == nil {
		log.Printf("[WARN] transcript session not found session=%s", sessionID)
		return
	}
	if englishText == "" {
		return
	}
	if isChineseText(englishText) {
		return
	}
	if speaker == "" {
		speaker = sess.GetCurrentSpeaker()
	}
	if speaker == "" {
		speaker = "A"
	}

	background := sess.GetBackgroundSummary()
	recentContext := sess.GetWindow(5)
	chineseText, err := p.translateForSession(sessionID, englishText, recentContext, background)
	if err != nil {

		log.Printf("[ERROR] translate failed session=%s err=%v", sessionID, err)
		chineseText = "[翻译失败]"
	}

	eventID := p.nextEventID()
	seg, shouldCorrect := sess.AppendSentence(englishText, chineseText, speaker)

	// Background context summarization: every 10 sentences, async refresh.
	if sess.NeedBackgroundSummary() {
		go p.refreshBackgroundSummary(sessionID, sess)
	}

	evt := sse.BuildSubtitleFinal(eventID, seg.Index, seg.Original, seg.Translation, speaker)
	p.broker.Publish(sessionID, evt)
	log.Printf("[sse] published subtitle.final session=%s segmentId=%d", sessionID, seg.Index)

	// Correction engine
	if shouldCorrect {
		window := sess.GetWindow(p.corr.WindowSize)
		lookbackStart := len(window) - p.corr.LookbackCount
		if lookbackStart < 0 {
			lookbackStart = 0
		}
		lookback := window[lookbackStart:]
		log.Printf("[corrector] triggered session=%s window=%d lookback=%d",
			sessionID, len(window), len(lookback))

		prompt := p.corr.BuildCorrectionPrompt(window, lookback, background)
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
					corrEvt := sse.BuildCorrection(p.nextEventID(), suggestion.SegmentIndex,
						oldText, suggestion.NewTranslation, newRev)
					p.broker.Publish(sessionID, corrEvt)
					log.Printf("[sse] published subtitle.corrected session=%s segmentId=%d rev=%d",
						sessionID, suggestion.SegmentIndex, newRev)
				}
			}
		}
	}

}

// applyDiarizationSpeaker retroactively sets the speaker label on the most
// recent subtitle segment that lacks one, then publishes a speaker update event.
func (p *Pipeline) applyDiarizationSpeaker(sessionID, speaker string) {
	if speaker == "" {
		return
	}
	sess := p.mgr.Get(sessionID)
	if sess == nil {
		return
	}
	sentences := sess.GetWindow(6)
	for i := len(sentences) - 1; i >= 0; i-- {
		if sentences[i].Speaker != speaker {
			if sess.ApplySpeaker(sentences[i].Index, speaker) {
				evt := sse.BuildSpeakerUpdate(p.nextEventID(), sentences[i].Index, speaker)
				p.broker.Publish(sessionID, evt)
				log.Printf("[diarization] speaker assigned session=%s segment=%d speaker=%s",
					sessionID, sentences[i].Index, speaker)
				sess.SetCurrentSpeaker(speaker)
			}
			return
		}
	}
}

func (p *Pipeline) handleFallbackAudio(sessionID string, pcm []byte) {
	sess := p.mgr.Get(sessionID)
	if sess == nil || len(pcm) == 0 {
		return
	}
	wavAudio := audio.WriteWAVHeader(48000, pcm)
	englishText, err := p.asr.Transcribe(wavAudio)
	if err != nil {
		return
	}
	if englishText == "" {
		return
	}
	if englishText == "" {
		return
	}
	speaker := sess.GetCurrentSpeaker()
	if speaker == "" {
		speaker = "A"
	}
	log.Printf("[diarization] fallback ASR subtitle session=%s speaker=%s", sessionID, speaker)
	p.handleFinalTranscript(sessionID, englishText, speaker)
}

// translateForSession translates a finalized transcript for one session.
func (p *Pipeline) translateForSession(sessionID, englishText string, recent []session.Sentence, background string) (string, error) {
	if !p.tts.Enabled(sessionID) {
		return p.transl.TranslateWithContext(englishText, recent, background)
	}

	chunker := p.ttsChunkerFor(sessionID)
	speakChunks := func(chunks []string) {
		for _, chunk := range chunks {
			p.tts.Speak(sessionID, chunk)
		}
	}

	result, err := p.transl.TranslateStreamWithContext(englishText, recent, background, func(delta string) {
		speakChunks(chunker.Add(delta, time.Now()))
	})
	if err != nil {
		return "", err
	}
	speakChunks(chunker.Ready(time.Now().Add(750 * time.Millisecond)))
	return result, nil
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
	p.diar.Stop(sessionID)
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
		p.removeTTSChunker(sessionID)
	}()

	log.Printf("[session] stopped id=%s dur=%s", sessionID, duration.Round(time.Second))
	w.Write([]byte(`{"status":"stopped"}`))
}

// HandleDebugCorrection injects test sentences and fires a correction for
// manual verification of the correction UI animation.
func (p *Pipeline) HandleDebugCorrection(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("sessionId")
	sess := p.mgr.Get(sessionID)
	if sess == nil {
		http.Error(w, `{"error":"session not found"}`, http.StatusNotFound)
		return
	}

	// Seed 6 varied sentences so the correction window has enough data.
	pairs := [][2]string{
		{"The transformer architecture revolutionized NLP.", "Transformer 架构革新了自然语言处理。"},
		{"It uses self-attention to process sequences.", "它使用自注意力来处理序列。"},
		{"The model computes attention weights.", "该模型计算注意力权重。"},
		{"Attention helps the model focus on relevant tokens.", "注意力帮助模型聚焦于相关标记。"},
		{"The decoder generates output tokens one by one.", "解码器逐一生成输出标记。"},
		{"This approach has become the foundation of modern AI.", "这种方法已经成为现代 AI 的基础。"},
	}
	for _, pair := range pairs {
		sess.AppendSentence(pair[0], pair[1], "A")
	}

	// Force a correction on the first sentence (segment index 0).
	oldText := pairs[0][1]
	newText := "Transformer 架构彻底改变了自然语言处理领域。"
	newRev := int(2)

	if sess.ApplyCorrection(0, newText, newRev) {
		p.eventMu.Lock()
		p.eventID++
		id := p.eventID
		p.eventMu.Unlock()
		corrEvt := sse.BuildCorrection(id, 0, oldText, newText, newRev)
		p.broker.Publish(sessionID, corrEvt)
		log.Printf("[debug] correction published session=%s segmentId=0 old=%q new=%q",
			sessionID, oldText, newText)
	}

	// Also publish the 6 subtitle.final events first so the client has them.
	p.eventMu.Lock()
	for i, pair := range pairs {
		p.eventID++
		evt := sse.BuildSubtitleFinal(p.eventID, i, pair[0], pair[1], "A")
		p.broker.Publish(sessionID, evt)
	}
	// Re-publish correction after subtitles to ensure it's processed after them.
	p.eventID++
	corrEvt2 := sse.BuildCorrection(p.eventID, 0, oldText, newText, newRev)
	p.broker.Publish(sessionID, corrEvt2)
	p.eventMu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok","message":"6 test sentences + correction sent"}`))
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

func (p *Pipeline) ttsChunkerFor(sessionID string) *tts.StreamChunker {
	p.ttsChunkersMu.Lock()
	defer p.ttsChunkersMu.Unlock()

	chunker := p.ttsChunkers[sessionID]
	if chunker == nil {
		chunker = tts.NewStreamChunker()
		p.ttsChunkers[sessionID] = chunker
	}
	return chunker
}

func (p *Pipeline) removeTTSChunker(sessionID string) {
	p.ttsChunkersMu.Lock()
	delete(p.ttsChunkers, sessionID)
	p.ttsChunkersMu.Unlock()
}

// refreshBackgroundSummary asynchronously asks the LLM to summarize the
// conversation domain, key terminology, and speaker perspective, then
// stores the result in the session for future translation prompts.
func (p *Pipeline) refreshBackgroundSummary(sessionID string, sess *session.Session) {
	sentences := sess.GetWindow(20)
	if len(sentences) < 10 {
		return
	}
	existing := sess.GetBackgroundSummary()
	summary, err := p.transl.SummarizeBackground(sentences, existing)
	if err != nil {
		log.Printf("[summarize] failed session=%s err=%v", sessionID, err)
		return
	}
	sess.SetBackgroundSummary(summary)
	log.Printf("[summarize] updated session=%s len=%d", sessionID, len(summary))

	p.eventMu.Lock()
	p.eventID++
	evt := sse.BuildBackgroundSummary(p.eventID, summary, len(sentences))
	p.eventMu.Unlock()
	p.broker.Publish(sessionID, evt)
	log.Printf("[sse] published background.summary session=%s", sessionID)
}

// isChineseText returns true when the text appears to be Chinese (likely TTS echo).
func isChineseText(text string) bool {
	runes := []rune(text)
	if len(runes) == 0 {
		return false
	}
	cjk := 0
	for _, r := range runes {
		if r >= 0x4E00 && r <= 0x9FFF {
			cjk++
		}
	}
	return float64(cjk)/float64(len(runes)) > 0.25
}
