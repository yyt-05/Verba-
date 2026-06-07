package tts

import (
	"fmt"
	"log"
	"strings"
	"sync"
)

type Config struct {
	Provider        string
	DashScopeAPIKey string
	RealtimeURL     string
	Model           string
	Voice           string
	Language        string
	OnAudio         func(sessionID string, audio []byte)
	OnReset         func(sessionID string)
}

type Manager struct {
	cfg      Config
	mu       sync.RWMutex
	sessions map[string]*Session
	pending  map[string]*Session
}

func NewManager(cfg Config) *Manager {
	return &Manager{
		cfg:      cfg,
		sessions: make(map[string]*Session),
		pending:  make(map[string]*Session),
	}
}

func (m *Manager) Enable(sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("missing session id")
	}
	if strings.ToLower(m.cfg.Provider) != "dashscope" {
		return fmt.Errorf("tts provider is not dashscope")
	}
	if m.cfg.DashScopeAPIKey == "" {
		return fmt.Errorf("DASHSCOPE_API_KEY is not configured")
	}

	m.mu.Lock()
	if existing := m.sessions[sessionID]; existing != nil {
		m.mu.Unlock()
		return nil
	}
	if existing := m.pending[sessionID]; existing != nil {
		m.mu.Unlock()
		return nil
	}

	session := NewSession(sessionID, m.cfg)
	m.pending[sessionID] = session
	m.mu.Unlock()

	go m.startSession(sessionID, session)

	log.Printf("[tts] enable requested session=%s provider=dashscope model=%s voice=%s",
		sessionID, m.cfg.Model, m.cfg.Voice)
	return nil
}

func (m *Manager) startSession(sessionID string, session *Session) {
	if err := session.Start(); err != nil {
		m.mu.Lock()
		if m.pending[sessionID] == session {
			delete(m.pending, sessionID)
		}
		m.mu.Unlock()
		log.Printf("[tts] start failed session=%s err=%v", sessionID, err)
		return
	}

	m.mu.Lock()
	if m.pending[sessionID] != session {
		m.mu.Unlock()
		session.Stop()
		return
	}
	delete(m.pending, sessionID)
	m.sessions[sessionID] = session
	m.mu.Unlock()

	log.Printf("[tts] enabled session=%s provider=dashscope model=%s voice=%s",
		sessionID, m.cfg.Model, m.cfg.Voice)
}

func (m *Manager) Disable(sessionID string) {
	m.mu.Lock()
	session := m.sessions[sessionID]
	pending := m.pending[sessionID]
	delete(m.sessions, sessionID)
	delete(m.pending, sessionID)
	m.mu.Unlock()

	if session != nil {
		session.Stop()
		log.Printf("[tts] disabled session=%s", sessionID)
	}
	if pending != nil {
		pending.Stop()
		log.Printf("[tts] canceled pending session=%s", sessionID)
	}
}

func (m *Manager) Enabled(sessionID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessions[sessionID] != nil
}

func (m *Manager) Speak(sessionID, text string) {
	m.mu.RLock()
	session := m.sessions[sessionID]
	m.mu.RUnlock()
	if session == nil {
		return
	}
	session.Enqueue(text)
}
