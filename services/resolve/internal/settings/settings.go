package settings

import "sync"

// Settings holds Resolve lab toggles (Phase 2).
type Settings struct {
	mu         sync.RWMutex
	DoHEnabled bool `json:"doh_enabled"`
}

func New() *Settings {
	return &Settings{DoHEnabled: true}
}

func (s *Settings) Get() Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return Settings{DoHEnabled: s.DoHEnabled}
}

func (s *Settings) SetDoH(enabled bool) {
	s.mu.Lock()
	s.DoHEnabled = enabled
	s.mu.Unlock()
}

func (s *Settings) DoH() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.DoHEnabled
}
