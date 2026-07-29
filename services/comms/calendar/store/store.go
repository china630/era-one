// Package store — in-memory CalDAV event store (MVP Wave C-2).
package store

import (
	"sync"
	"time"
)

// Event is a calendar object (iCal VEVENT body).
type Event struct {
	UID       string
	Owner     string
	Body      string
	UpdatedAt time.Time
}

// Store holds per-user calendar events keyed by UID.
type Store struct {
	mu     sync.RWMutex
	events map[string]map[string]*Event
}

// New returns an empty calendar store.
func New() *Store {
	return &Store{events: make(map[string]map[string]*Event)}
}

// Put creates or replaces an event.
func (s *Store) Put(owner, uid, body string) *Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.events[owner] == nil {
		s.events[owner] = make(map[string]*Event)
	}
	ev := &Event{
		UID:       uid,
		Owner:     owner,
		Body:      body,
		UpdatedAt: time.Now().UTC(),
	}
	s.events[owner][uid] = ev
	return ev
}

// Get returns event by owner and UID.
func (s *Store) Get(owner, uid string) (*Event, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if m := s.events[owner]; m != nil {
		ev, ok := m[uid]
		return ev, ok
	}
	return nil, false
}

// List returns all events for owner.
func (s *Store) List(owner string) []*Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m := s.events[owner]
	out := make([]*Event, 0, len(m))
	for _, ev := range m {
		out = append(out, ev)
	}
	return out
}

// FindByUID returns the first event matching uid across all owners.
func (s *Store) FindByUID(uid string) (*Event, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, events := range s.events {
		if ev, ok := events[uid]; ok {
			return ev, true
		}
	}
	return nil, false
}
