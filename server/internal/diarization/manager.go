package diarization

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type stream interface {
	SendPCM([]byte) error
	Close()
}

type Manager struct {
	cfg      Config
	streams  map[string]stream
	speakers map[string]string // sessionID -> current speaker label
	mu       sync.Mutex
}

func NewManager(cfg Config) *Manager {
	return &Manager{
		cfg:      cfg,
		streams:  make(map[string]stream),
		speakers: make(map[string]string),
	}
}

func (m *Manager) Enabled() bool {
	switch strings.ToLower(m.cfg.Provider) {
	case "deepgram":
		return m.cfg.APIKey != ""
	case "tencent":
		return m.cfg.AppID != "" && m.cfg.SecretID != "" && m.cfg.SecretKey != ""
	default:
		return false
	}
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
	case "tencent":
		s, err := newTencentStream(m.cfg, sessionID)
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

	// Split large chunks to avoid overwhelming the diarization API.
	const maxChunkBytes = 2 * 48000 * 2 // 2 seconds of 48kHz PCM16
	for offset := 0; offset < len(pcm); offset += maxChunkBytes {
		end := offset + maxChunkBytes
		if end > len(pcm) {
			end = len(pcm)
		}
		chunk := pcm[offset:end]
		if err := s.SendPCM(chunk); err != nil {
			m.mu.Lock()
			if m.streams[sessionID] == s {
				s.Close()
				delete(m.streams, sessionID)
			}
			m.mu.Unlock()
			return err
		}
		if end < len(pcm) {
			time.Sleep(200 * time.Millisecond)
		}
	}
	return nil
}

// SetSpeaker records the current speaker label from the diarization provider.
func (m *Manager) SetSpeaker(sessionID, speaker string) {
	m.mu.Lock()
	m.speakers[sessionID] = speaker
	m.mu.Unlock()
}

// GetSpeaker returns the most recent speaker label, or "" if unknown.
func (m *Manager) GetSpeaker(sessionID string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.speakers[sessionID]
}

func (m *Manager) Stop(sessionID string) {
	m.mu.Lock()
	s := m.streams[sessionID]
	delete(m.streams, sessionID)
	delete(m.speakers, sessionID)
	m.mu.Unlock()
	if s != nil {
		s.Close()
	}
}
