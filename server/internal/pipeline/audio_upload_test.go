package pipeline

import (
	"net/http/httptest"
	"testing"

	"github.com/verba/server/internal/config"
	"github.com/verba/server/internal/session"
	"github.com/verba/server/internal/sse"
)

func TestAudioUpload_AccumulatesChunks(t *testing.T) {
	cfg := &config.Config{MaxSessionMin: 60}
	mgr := session.NewManager(cfg)
	s := mgr.Create("sess_accum")

	if len(s.GetWindow(100)) != 0 {
		t.Fatal("session should start with 0 sentences")
	}
}

func TestAudioUpload_MultipleChunksProduceMultipleSubtitles(t *testing.T) {
	cfg := &config.Config{MaxSessionMin: 60}
	mgr := session.NewManager(cfg)
	brk := sse.NewBroker()
	pipe := New(cfg, mgr, brk)

	mgr.Create("sess_multi")
	ch := brk.Subscribe("sess_multi")
	defer brk.Unsubscribe("sess_multi", ch)

	for i := range 5 {
		if rec := uploadSpeechSegment(pipe, "sess_multi"); rec.Code != 202 {
			t.Fatalf("chunk %d: expected 202, got %d", i, rec.Code)
		}
	}

	s := mgr.Get("sess_multi")
	window := s.GetWindow(100)
	if len(window) != 5 {
		t.Fatalf("expected 5 sentences after 5 uploads, got %d", len(window))
	}
	for i, seg := range window {
		if seg.Index != i {
			t.Fatalf("expected index %d, got %d", i, seg.Index)
		}
	}
}

func TestAudioUpload_CorrectionTriggersAt6(t *testing.T) {
	cfg := &config.Config{MaxSessionMin: 60}
	mgr := session.NewManager(cfg)
	brk := sse.NewBroker()
	pipe := New(cfg, mgr, brk)

	mgr.Create("sess_correct")
	ch := brk.Subscribe("sess_correct")
	defer brk.Unsubscribe("sess_correct", ch)

	for i := range 6 {
		if rec := uploadSpeechSegment(pipe, "sess_correct"); rec.Code != 202 {
			t.Fatalf("chunk %d: expected 202, got %d", i, rec.Code)
		}
	}

	gotCorrection := false
	gotSubtitle := false
	drained := false
	for !drained {
		select {
		case evt := <-ch:
			if evt.Type == sse.EventSubtitleFinal {
				gotSubtitle = true
			}
			if evt.Type == sse.EventSubtitleCorrected {
				gotCorrection = true
			}
		default:
			drained = true
		}
	}

	if !gotSubtitle {
		t.Fatal("expected at least one subtitle event")
	}
	if !gotCorrection {
		t.Fatal("expected at least one correction event when 6 sentences reached")
	}
}

func TestAudioUpload_SessionIsolation(t *testing.T) {
	cfg := &config.Config{MaxSessionMin: 60}
	mgr := session.NewManager(cfg)
	brk := sse.NewBroker()
	pipe := New(cfg, mgr, brk)

	mgr.Create("sess_A")
	mgr.Create("sess_B")
	chB := brk.Subscribe("sess_B")
	defer brk.Unsubscribe("sess_B", chB)

	uploadSpeechSegment(pipe, "sess_A")

	select {
	case <-chB:
		t.Fatal("session B should not receive events from session A")
	default:
	}

	if len(mgr.Get("sess_A").GetWindow(10)) != 1 {
		t.Fatal("session A should have 1 sentence")
	}
	if len(mgr.Get("sess_B").GetWindow(10)) != 0 {
		t.Fatal("session B should have 0 sentences")
	}
}

func TestAudioUpload_RejectsAfterStop(t *testing.T) {
	cfg := &config.Config{MaxSessionMin: 60}
	mgr := session.NewManager(cfg)
	brk := sse.NewBroker()
	pipe := New(cfg, mgr, brk)

	mgr.Create("sess_stopped")
	mgr.Get("sess_stopped").SetStatus(session.StatusStopped)

	rec := uploadAudioChunk(pipe, "sess_stopped", testPCM(300, 2000))

	// Phase 0: upload to stopped still works (status check added in Phase 3)
	if rec.Code != 202 {
		t.Fatalf("Phase 0: upload to stopped session should still work, got %d", rec.Code)
	}
}

func uploadSpeechSegment(pipe *Pipeline, sessionID string) *httptest.ResponseRecorder {
	var rec *httptest.ResponseRecorder
	for _, chunk := range [][]byte{
		testPCM(300, 2000),
		testPCM(300, 2000),
		testPCM(300, 2000),
		testPCM(800, 0),
	} {
		rec = uploadAudioChunk(pipe, sessionID, chunk)
		if rec.Code != 202 {
			return rec
		}
	}
	return rec
}
