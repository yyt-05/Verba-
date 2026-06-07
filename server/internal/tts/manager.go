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
}

type Manager struct {
	cfg      Config
	player   *PCMPlayer
	mu       sync.RWMutex
	sessions map[string]*Session
}

func NewManager(cfg Config) *Manager {
	return &Manager{
		cfg:      cfg,
		player:   NewPCMPlayer(),
		sessions: make(map[string]*Session),
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
	defer m.mu.Unlock()
	if existing := m.sessions[sessionID]; existing != nil {
		return nil
	}

	session := NewSession(sessionID, m.cfg, m.player)
	m.sessions[sessionID] = session
	session.Start()
	log.Printf("[tts] enabled session=%s provider=dashscope model=%s voice=%s",
		sessionID, m.cfg.Model, m.cfg.Voice)
	return nil
}

func (m *Manager) Disable(sessionID string) {
	m.mu.Lock()
	session := m.sessions[sessionID]
	delete(m.sessions, sessionID)
	m.mu.Unlock()

	if session != nil {
		session.Stop()
		log.Printf("[tts] disabled session=%s", sessionID)
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
