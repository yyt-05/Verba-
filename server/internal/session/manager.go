package session

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/verba/server/internal/config"
)

// Status represents the current session state.
type Status string

const (
	StatusCreated   Status = "created"
	StatusListening Status = "listening"
	StatusSilent    Status = "silent"
	StatusStopped   Status = "stopped"
	StatusError     Status = "error"
)

// Sentence holds one subtitle segment.
type Sentence struct {
	Index       int       // global increment within session
	Original    string    // English ASR text
	Translation string    // Chinese translation
	Revision    int       // starts at 1, incremented on correction
	CreatedAt   time.Time
}

// Session represents one user session.
type Session struct {
	ID        string
	Status    Status
	CreatedAt time.Time
	Sentences []Sentence
	Seq       int // next sentence index

	// Correction window state
	WindowHash string // hash of the current 12-sentence window

	mu     sync.RWMutex
	cancel context.CancelFunc
}

// Manager holds all active sessions.
type Manager struct {
	cfg      *config.Config
	sessions map[string]*Session
	mu       sync.RWMutex
}

func NewManager(cfg *config.Config) *Manager {
	return &Manager{
		cfg:      cfg,
		sessions: make(map[string]*Session),
	}
}

func (m *Manager) Create(id string) *Session {
	s := &Session{
		ID:        id,
		Status:    StatusCreated,
		CreatedAt: time.Now(),
		Sentences: make([]Sentence, 0),
		Seq:       0,
	}
	m.mu.Lock()
	m.sessions[id] = s
	m.mu.Unlock()
	log.Printf("[session] created: %s", id)
	return s
}

func (m *Manager) Get(id string) *Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessions[id]
}

func (m *Manager) Remove(id string) {
	m.mu.Lock()
	delete(m.sessions, id)
	m.mu.Unlock()
	log.Printf("[session] removed: %s", id)
}

// AppendSentence adds a new sentence and returns whether correction should be triggered.
func (s *Session) AppendSentence(original, translation string) (sentence Sentence, shouldCorrect bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Status = StatusListening
	seg := Sentence{
		Index:       s.Seq,
		Original:    original,
		Translation: translation,
		Revision:    1,
		CreatedAt:   time.Now(),
	}
	s.Sentences = append(s.Sentences, seg)
	s.Seq++

	// Trigger correction every 3 new sentences, when window has ≥6 sentences
	shouldCorrect = len(s.Sentences) >= 6 && s.Seq%3 == 0

	return seg, shouldCorrect
}

// GetWindow returns the last N sentences for the correction context window.
func (s *Session) GetWindow(n int) []Sentence {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.Sentences) <= n {
		result := make([]Sentence, len(s.Sentences))
		copy(result, s.Sentences)
		return result
	}
	start := len(s.Sentences) - n
	result := make([]Sentence, n)
	copy(result, s.Sentences[start:])
	return result
}

// ApplyCorrection updates a sentence at the given index if the revision is newer.
func (s *Session) ApplyCorrection(index int, newTranslation string, revision int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.Sentences {
		if s.Sentences[i].Index == index {
			if revision > s.Sentences[i].Revision {
				s.Sentences[i].Translation = newTranslation
				s.Sentences[i].Revision = revision
				return true
			}
			return false
		}
	}
	return false
}

func (s *Session) SetStatus(status Status) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = status
}
