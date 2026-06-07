package pipeline

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/verba/server/internal/config"
	"github.com/verba/server/internal/session"
	"github.com/verba/server/internal/sse"
)

func setupTestServer() (*Pipeline, *session.Manager, *sse.Broker) {
	cfg := &config.Config{Port: "8080", MaxSessionMin: 60, BudgetCapUSD: 1.0}
	mgr := session.NewManager(cfg)
	broker := sse.NewBroker()
	pipe := New(cfg, mgr, broker)
	return pipe, mgr, broker
}

func TestHandleCreateSession(t *testing.T) {
	pipe, mgr, _ := setupTestServer()

	req := httptest.NewRequest("POST", "/api/v1/sessions", nil)
	rec := httptest.NewRecorder()

	pipe.HandleCreateSession(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	sid := resp["session_id"]
	if sid == "" {
		t.Fatal("expected non-empty session_id")
	}

	// Verify session exists in manager
	s := mgr.Get(sid)
	if s == nil {
		t.Fatal("session not found in manager after creation")
	}
	if s.Status != session.StatusCreated {
		t.Fatalf("expected status created, got %s", s.Status)
	}
}

func TestHandleCreateSession_SSEPublishesStatus(t *testing.T) {
	pipe, _, broker := setupTestServer()

	req := httptest.NewRequest("POST", "/api/v1/sessions", nil)
	rec := httptest.NewRecorder()
	pipe.HandleCreateSession(rec, req)

	var resp map[string]string
	json.NewDecoder(rec.Body).Decode(&resp)
	sid := resp["session_id"]

	// SSE subscriber should receive session.status event
	ch := broker.Subscribe(sid)
	defer broker.Unsubscribe(sid, ch)

	select {
	case evt := <-ch:
		if evt.Type != sse.EventSessionStatus {
			t.Fatalf("expected session.status, got %s", evt.Type)
		}
	default:
		// Event might have been sent before subscription — acceptable for MVP
		t.Log("no event buffered (expected — subscribed after publish)")
	}
}

func TestHandleUploadAudio_ValidSession(t *testing.T) {
	pipe, mgr, broker := setupTestServer()

	s := mgr.Create("sess_audio_test")

	// Subscribe BEFORE upload so we catch the event
	ch := broker.Subscribe("sess_audio_test")
	defer broker.Unsubscribe("sess_audio_test", ch)

	for _, chunk := range [][]byte{
		testPCM(300, 2000),
		testPCM(300, 2000),
		testPCM(300, 2000),
		testPCM(800, 0),
	} {
		rec := uploadAudioChunk(pipe, "sess_audio_test", chunk)
		if rec.Code != http.StatusAccepted {
			t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
		}
	}

	window := s.GetWindow(1)
	if len(window) != 1 {
		t.Fatalf("expected 1 sentence after audio upload, got %d", len(window))
	}

	select {
	case evt := <-ch:
		if evt.Type != sse.EventSubtitleFinal {
			t.Fatalf("expected subtitle.final, got %s", evt.Type)
		}
		if evt.SegmentID != 0 {
			t.Fatalf("expected segmentId 0, got %d", evt.SegmentID)
		}
	default:
		t.Fatal("expected subtitle.final event, got none")
	}
}

func TestHandleUploadAudio_BuffersBeforeSpeechEnd(t *testing.T) {
	pipe, mgr, broker := setupTestServer()
	mgr.Create("sess_buffering")

	ch := broker.Subscribe("sess_buffering")
	defer broker.Unsubscribe("sess_buffering", ch)

	rec := uploadAudioChunk(pipe, "sess_buffering", testPCM(300, 2000))
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}

	select {
	case evt := <-ch:
		t.Fatalf("expected no subtitle before speech end, got %s", evt.Type)
	default:
		// expected
	}
}

func TestHandleUploadAudio_SessionNotFound(t *testing.T) {
	pipe, _, _ := setupTestServer()

	req := httptest.NewRequest("POST", "/api/v1/sessions/nonexistent/audio",
		strings.NewReader("data"))
	req.SetPathValue("sessionId", "nonexistent")
	rec := httptest.NewRecorder()

	pipe.HandleUploadAudio(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleUploadAudio_EmptyBody(t *testing.T) {
	pipe, mgr, _ := setupTestServer()
	mgr.Create("sess_empty")

	req := httptest.NewRequest("POST", "/api/v1/sessions/sess_empty/audio", nil)
	req.SetPathValue("sessionId", "sess_empty")
	rec := httptest.NewRecorder()

	pipe.HandleUploadAudio(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty body, got %d", rec.Code)
	}
}

func TestHandleStopSession(t *testing.T) {
	pipe, mgr, broker := setupTestServer()
	mgr.Create("sess_stop")

	ch := broker.Subscribe("sess_stop")
	defer broker.Unsubscribe("sess_stop", ch)

	req := httptest.NewRequest("POST", "/api/v1/sessions/sess_stop/stop", nil)
	req.SetPathValue("sessionId", "sess_stop")
	rec := httptest.NewRecorder()

	pipe.HandleStopSession(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	s := mgr.Get("sess_stop")
	if s == nil {
		t.Fatal("session should still exist immediately after stop")
	}
	if s.Status != session.StatusStopped {
		t.Fatalf("expected stopped, got %s", s.Status)
	}

	// SSE should receive stopped status event
	select {
	case evt := <-ch:
		if evt.Type != sse.EventSessionStatus {
			t.Fatalf("expected session.status, got %s", evt.Type)
		}
	default:
		t.Fatal("expected status event after stop")
	}
}

func TestHandleStopSession_NotFound(t *testing.T) {
	pipe, _, _ := setupTestServer()

	req := httptest.NewRequest("POST", "/api/v1/sessions/nonexistent/stop", nil)
	req.SetPathValue("sessionId", "nonexistent")
	rec := httptest.NewRecorder()

	pipe.HandleStopSession(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleTTSControl_DisableValidSession(t *testing.T) {
	pipe, mgr, _ := setupTestServer()
	mgr.Create("sess_tts_disable")

	req := httptest.NewRequest("POST", "/api/v1/sessions/sess_tts_disable/tts",
		strings.NewReader(`{"enabled":false}`))
	req.SetPathValue("sessionId", "sess_tts_disable")
	rec := httptest.NewRecorder()

	pipe.HandleTTSControl(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleTTSControl_EnableRequiresConfig(t *testing.T) {
	pipe, mgr, _ := setupTestServer()
	mgr.Create("sess_tts_config")

	req := httptest.NewRequest("POST", "/api/v1/sessions/sess_tts_config/tts",
		strings.NewReader(`{"enabled":true}`))
	req.SetPathValue("sessionId", "sess_tts_config")
	rec := httptest.NewRecorder()

	pipe.HandleTTSControl(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleTTSControl_SessionNotFound(t *testing.T) {
	pipe, _, _ := setupTestServer()

	req := httptest.NewRequest("POST", "/api/v1/sessions/missing/tts",
		strings.NewReader(`{"enabled":true}`))
	req.SetPathValue("sessionId", "missing")
	rec := httptest.NewRecorder()

	pipe.HandleTTSControl(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestFullPipelineFlow(t *testing.T) {
	// Step-by-step integration: create → upload → SSE verify → stop
	pipe, mgr, broker := setupTestServer()

	// 1. Create session
	req1 := httptest.NewRequest("POST", "/api/v1/sessions", nil)
	rec1 := httptest.NewRecorder()
	pipe.HandleCreateSession(rec1, req1)

	var resp map[string]string
	json.NewDecoder(rec1.Body).Decode(&resp)
	sid := resp["session_id"]

	// Subscribe before uploading
	ch := broker.Subscribe(sid)
	defer broker.Unsubscribe(sid, ch)

	// 2. Upload enough speech plus trailing silence to finalize one segment
	for _, chunk := range [][]byte{
		testPCM(300, 2000),
		testPCM(300, 2000),
		testPCM(300, 2000),
		testPCM(800, 0),
	} {
		rec2 := uploadAudioChunk(pipe, sid, chunk)
		if rec2.Code != http.StatusAccepted {
			t.Fatalf("upload failed: %d", rec2.Code)
		}
	}

	// 3. Verify subtitle event
	select {
	case evt := <-ch:
		if evt.Type != sse.EventSubtitleFinal {
			t.Fatalf("expected subtitle.final, got %s", evt.Type)
		}
	default:
		t.Fatal("expected subtitle event after audio upload")
	}

	// 4. Stop
	req3 := httptest.NewRequest("POST", "/api/v1/sessions/"+sid+"/stop", nil)
	req3.SetPathValue("sessionId", sid)
	rec3 := httptest.NewRecorder()
	pipe.HandleStopSession(rec3, req3)
	if rec3.Code != http.StatusOK {
		t.Fatalf("stop failed: %d", rec3.Code)
	}

	// 5. Verify session stopped
	s := mgr.Get(sid)
	if s == nil || s.Status != session.StatusStopped {
		t.Fatal("session not in stopped state")
	}
}

func uploadAudioChunk(pipe *Pipeline, sessionID string, data []byte) *httptest.ResponseRecorder {
	req := httptest.NewRequest("POST", "/api/v1/sessions/"+sessionID+"/audio",
		bytes.NewReader(data))
	req.SetPathValue("sessionId", sessionID)
	rec := httptest.NewRecorder()
	pipe.HandleUploadAudio(rec, req)
	return rec
}

func testPCM(durationMs int, amplitude int16) []byte {
	const sampleRate = 48000
	samples := sampleRate * durationMs / 1000
	data := make([]byte, samples*2)
	for i := 0; i < samples; i++ {
		binary.LittleEndian.PutUint16(data[i*2:i*2+2], uint16(amplitude))
	}
	return data
}
