package atlas

import (
	"encoding/json"
	"os"
	"strings"
	"sync"
)

// Entry is a domain IoC in an offline pack.
type Entry struct {
	Domain   string `json:"domain"`
	Severity string `json:"severity"`
	Source   string `json:"source,omitempty"`
}

// Pack is an offline Atlas TI pack.
type Pack struct {
	ID      string  `json:"id"`
	Version string  `json:"version"`
	Domains []Entry `json:"domains"`
}

// Store indexes Atlas domains for Guard.
type Store struct {
	mu   sync.RWMutex
	by   map[string]Entry
	meta Pack
}

func New() *Store {
	return &Store{by: map[string]Entry{}}
}

func (s *Store) LoadFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var p Pack
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	return s.Load(p)
}

func (s *Store) Load(p Pack) error {
	idx := make(map[string]Entry, len(p.Domains))
	for _, e := range p.Domains {
		d := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(e.Domain)), ".")
		if d != "" {
			idx[d] = e
		}
	}
	s.mu.Lock()
	s.by = idx
	s.meta = p
	s.mu.Unlock()
	return nil
}

func (s *Store) Meta() Pack {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.meta
}

// Clear removes the loaded pack (named packs only — id must match when non-empty).
func (s *Store) Clear(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.meta.ID == "" && len(s.by) == 0 {
		return false
	}
	if id != "" && s.meta.ID != "" && s.meta.ID != id {
		return false
	}
	s.by = map[string]Entry{}
	s.meta = Pack{}
	return true
}

// Lookup returns an Atlas hit for qname.
func (s *Store) Lookup(qname string) (Entry, bool) {
	name := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(qname)), ".")
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.by[name]
	return e, ok
}
