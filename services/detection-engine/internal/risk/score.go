// Package risk — entity risk score и дедупликация алертов (S6-7).
package risk

import (
	"sync"
	"time"
)

// Scorer накапливает риск по сущности (node) и подавляет повторы.
type Scorer struct {
	mu      sync.Mutex
	window  time.Duration
	dedup   map[string]time.Time
	scores  map[string]float64
	weights map[string]float64
}

func New(window time.Duration) *Scorer {
	return &Scorer{
		window: window,
		dedup:  make(map[string]time.Time),
		scores: make(map[string]float64),
		weights: map[string]float64{
			"critical": 25,
			"high":     15,
			"medium":   8,
			"low":      3,
			"info":     1,
		},
	}
}

// ShouldEmit возвращает false, если тот же rule+node уже срабатывал в окне dedup.
func (s *Scorer) ShouldEmit(ruleID, nodeID string, at time.Time) bool {
	key := ruleID + "|" + nodeID
	s.mu.Lock()
	defer s.mu.Unlock()
	cutoff := at.Add(-s.window)
	for k, ts := range s.dedup {
		if ts.Before(cutoff) {
			delete(s.dedup, k)
		}
	}
	if last, ok := s.dedup[key]; ok && last.After(cutoff) {
		return false
	}
	s.dedup[key] = at
	return true
}

// Bump увеличивает risk score сущности по severity.
func (s *Scorer) Bump(nodeID, severity string, at time.Time) float64 {
	w := s.weights[severity]
	if w == 0 {
		w = 5
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.scores[nodeID] += w
	// decay старых очков не делаем в MVP — окно dedup отдельно
	return s.scores[nodeID]
}

// Score возвращает текущий risk score узла.
func (s *Scorer) Score(nodeID string) float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.scores[nodeID]
}
