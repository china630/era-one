// Package privilegedsession — аудит привилегированных сессий (DLP/PAM, ADR-0013).
package privilegedsession

import (
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Record struct {
	ID        string    `json:"id"`
	User      string    `json:"user"`
	Host      string    `json:"host"`
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at,omitempty"`
	Commands  []string  `json:"commands,omitempty"`
	Alerted   bool      `json:"alerted"`
}

type Alert struct {
	SessionID string    `json:"session_id"`
	Reason    string    `json:"reason"`
	At        time.Time `json:"at"`
}

type Store struct {
	mu       sync.Mutex
	sessions map[string]*Record
	alerts   []Alert
}

func NewStore() *Store {
	return &Store{sessions: make(map[string]*Record)}
}

func (s *Store) Start(user, host string) *Record {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := &Record{
		ID: uuid.NewString(), User: user, Host: host, StartedAt: time.Now().UTC(),
	}
	s.sessions[r.ID] = r
	return r
}

// EnsureSession создаёт запись с заданным ID (SSH proxy, P-02).
func (s *Store) EnsureSession(sessionID, user, host string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.sessions[sessionID]; ok {
		return
	}
	s.sessions[sessionID] = &Record{
		ID: sessionID, User: user, Host: host, StartedAt: time.Now().UTC(),
	}
}

func (s *Store) LogCommand(sessionID, cmd string) (*Alert, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.sessions[sessionID]
	if !ok {
		return nil, false
	}
	rec.Commands = append(rec.Commands, cmd)
	if suspicious(cmd) {
		rec.Alerted = true
		a := Alert{SessionID: sessionID, Reason: "suspicious privileged command: " + cmd, At: time.Now().UTC()}
		s.alerts = append(s.alerts, a)
		return &a, true
	}
	return nil, false
}

func (s *Store) End(sessionID string) (*Record, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.sessions[sessionID]
	if !ok {
		return nil, false
	}
	rec.EndedAt = time.Now().UTC()
	return rec, true
}

func (s *Store) List() []*Record {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*Record, 0, len(s.sessions))
	for _, r := range s.sessions {
		out = append(out, r)
	}
	return out
}

func (s *Store) Alerts() []Alert {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Alert, len(s.alerts))
	copy(out, s.alerts)
	return out
}

func suspicious(cmd string) bool {
	low := strings.ToLower(cmd)
	for _, p := range []string{"curl ", "wget ", "scp ", "base64 -d", "certutil", "mimikatz", "/etc/shadow"} {
		if strings.Contains(low, p) {
			return true
		}
	}
	return false
}
