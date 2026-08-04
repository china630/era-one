package recording

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Event is protocol metadata (not Guacamole video).
type Event struct {
	SessionID string    `json:"session_id"`
	Kind      string    `json:"kind"` // connect | clipboard | keyframe_stub | disconnect
	Detail    string    `json:"detail,omitempty"`
	At        time.Time `json:"at"`
}

// Artifact is a session recording metadata file + hash.
type Artifact struct {
	SessionID string `json:"session_id"`
	Path      string `json:"path"`
	SHA256    string `json:"sha256"`
	Events    int    `json:"events"`
}

// Store accumulates metadata events and flushes artifacts.
type Store struct {
	mu   sync.Mutex
	dir  string
	ev   map[string][]Event
	arts map[string]*Artifact
}

func NewStore(dir string) *Store {
	_ = os.MkdirAll(dir, 0o700)
	return &Store{dir: dir, ev: map[string][]Event{}, arts: map[string]*Artifact{}}
}

func (s *Store) Add(sessionID, kind, detail string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ev[sessionID] = append(s.ev[sessionID], Event{
		SessionID: sessionID, Kind: kind, Detail: detail, At: time.Now().UTC(),
	})
}

func (s *Store) Events(sessionID string) []Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := append([]Event(nil), s.ev[sessionID]...)
	return out
}

func (s *Store) Finalize(sessionID string) (*Artifact, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	events := s.ev[sessionID]
	data, err := json.MarshalIndent(events, "", "  ")
	if err != nil {
		return nil, err
	}
	path := filepath.Join(s.dir, sessionID+".json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return nil, err
	}
	sum := sha256.Sum256(data)
	art := &Artifact{
		SessionID: sessionID, Path: path,
		SHA256: hex.EncodeToString(sum[:]), Events: len(events),
	}
	s.arts[sessionID] = art
	return art, nil
}

func (s *Store) Artifact(sessionID string) (*Artifact, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.arts[sessionID]
	return a, ok
}
