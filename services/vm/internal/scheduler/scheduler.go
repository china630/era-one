// Package scheduler — планировщик VM-сканов (S7-2).
package scheduler

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// Job — запланированное сканирование.
type Job struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Targets     []string  `json:"targets"`
	CronExpr    string    `json:"cron_expr"`
	NextRun     time.Time `json:"next_run"`
	Enabled     bool      `json:"enabled"`
	LastRun     time.Time `json:"last_run,omitempty"`
	LastStatus  string    `json:"last_status,omitempty"`
	Concurrency int       `json:"concurrency"`
}

// Scheduler хранит jobs in-memory (prod hook — persistent store позже).
type Scheduler struct {
	mu   sync.RWMutex
	jobs map[string]*Job
}

func New() *Scheduler {
	return &Scheduler{jobs: make(map[string]*Job)}
}

// Create добавляет job; cronExpr — упрощённый интервал вида `@every 24h`.
func (s *Scheduler) Create(name string, targets []string, cronExpr string, concurrency int) *Job {
	s.mu.Lock()
	defer s.mu.Unlock()
	if concurrency <= 0 {
		concurrency = 5
	}
	j := &Job{
		ID: uuid.NewString(), Name: name, Targets: targets, CronExpr: cronExpr,
		NextRun: time.Now().UTC().Add(parseEvery(cronExpr)), Enabled: true, Concurrency: concurrency,
	}
	s.jobs[j.ID] = j
	return j
}

// List возвращает все jobs.
func (s *Scheduler) List() []*Job {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Job, 0, len(s.jobs))
	for _, j := range s.jobs {
		cp := *j
		out = append(out, &cp)
	}
	return out
}

// Get возвращает job по id.
func (s *Scheduler) Get(id string) (*Job, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	j, ok := s.jobs[id]
	if !ok {
		return nil, false
	}
	cp := *j
	return &cp, true
}

// MarkRun обновляет last_run/status.
func (s *Scheduler) MarkRun(id, status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if j, ok := s.jobs[id]; ok {
		j.LastRun = time.Now().UTC()
		j.LastStatus = status
		j.NextRun = j.LastRun.Add(parseEvery(j.CronExpr))
	}
}

func parseEvery(expr string) time.Duration {
	if expr == "" {
		return 24 * time.Hour
	}
	if d, err := time.ParseDuration(expr); err == nil {
		return d
	}
	if len(expr) > 6 && expr[:6] == "@every" {
		if d, err := time.ParseDuration(expr[6:]); err == nil {
			return d
		}
	}
	return 24 * time.Hour
}
