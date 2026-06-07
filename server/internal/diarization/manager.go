package diarization

import (
	"fmt"
	"strings"
	"sync"
)

type stream interface {
	SendPCM([]byte) error
	Close()
}

type Manager struct {
	cfg     Config
	streams map[string]stream
	mu      sync.Mutex
}

func NewManager(cfg Config) *Manager {
	return &Manager{
		cfg:     cfg,
		streams: make(map[string]stream),
	}
}

func (m *Manager) Enabled() bool {
	return strings.EqualFold(m.cfg.Provider, "deepgram") && m.cfg.APIKey != ""
}

func (m *Manager) Start(sessionID string) error {
	if !m.Enabled() {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.streams[sessionID] != nil {
		return nil
	}

	switch strings.ToLower(m.cfg.Provider) {
	case "deepgram":
		s, err := newDeepgramStream(m.cfg, sessionID)
		if err != nil {
			return err
		}
		m.streams[sessionID] = s
		return nil
	default:
		return fmt.Errorf("unsupported diarization provider: %s", m.cfg.Provider)
	}
}

func (m *Manager) AddPCM(sessionID string, pcm []byte) error {
	if !m.Enabled() {
		return nil
	}

	m.mu.Lock()
	s := m.streams[sessionID]
	m.mu.Unlock()
	if s == nil {
		if err := m.Start(sessionID); err != nil {
			return err
		}
		m.mu.Lock()
		s = m.streams[sessionID]
		m.mu.Unlock()
	}
	if s == nil {
		return nil
	}
	return s.SendPCM(pcm)
}

func (m *Manager) Stop(sessionID string) {
	m.mu.Lock()
	s := m.streams[sessionID]
	delete(m.streams, sessionID)
	m.mu.Unlock()
	if s != nil {
		s.Close()
	}
}
