package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/verba/server/internal/config"
	"github.com/verba/server/internal/pipeline"
	"github.com/verba/server/internal/session"
	"github.com/verba/server/internal/sse"
)

func main() {
	cfg := config.Load()

	mgr := session.NewManager(cfg)
	broker := sse.NewBroker()
	pipe := pipeline.New(cfg, mgr, broker)

	mux := http.NewServeMux()

	// POST /api/v1/sessions — create new session, returns session_id
	mux.HandleFunc("POST /api/v1/sessions", pipe.HandleCreateSession)

	// POST /api/v1/sessions/{sessionId}/audio — upload audio chunk
	mux.HandleFunc("POST /api/v1/sessions/{sessionId}/audio", pipe.HandleUploadAudio)

	// GET /api/v1/sessions/{sessionId}/events — SSE event stream
	mux.HandleFunc("GET /api/v1/sessions/{sessionId}/events", broker.HandleSSE)

	// POST /api/v1/sessions/{sessionId}/tts — toggle realtime TTS
	mux.HandleFunc("POST /api/v1/sessions/{sessionId}/tts", pipe.HandleTTSControl)

	// POST /api/v1/sessions/{sessionId}/stop — stop session
	mux.HandleFunc("POST /api/v1/sessions/{sessionId}/stop", pipe.HandleStopSession)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      withCORS(mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 0, // SSE requires no write timeout
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("[server] shutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()

	log.Printf("[server] Verba server starting on :%s", cfg.Port)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("[server] fatal: %v", err)
	}
	log.Println("[server] stopped")
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
