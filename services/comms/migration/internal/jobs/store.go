package jobs

import (
	"sync"
	"time"
)

type Job struct {
	ID         string    `json:"id"`
	Source     string    `json:"source"`
	Mailbox    string    `json:"mailbox"`
	Status     string    `json:"status"`
	ItemsTotal int       `json:"items_total"`
	ItemsOK    int       `json:"items_ok"`
	ItemsFail  int       `json:"items_fail"`
	Error      string    `json:"error,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type Store struct {
	mu    sync.RWMutex
	jobs  map[string]Job
	seen  map[string]bool
	seqNo int
}

func NewStore() *Store {
	return &Store{
		jobs: make(map[string]Job),
		seen: make(map[string]bool),
	}
}

func (s *Store) CreateQueued(source, mailbox string) Job {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seqNo++
	id := time.Now().UTC().Format("20060102150405") + "-mig-" + source
	j := Job{
		ID:        id,
		Source:    source,
		Mailbox:   mailbox,
		Status:    "queued",
		CreatedAt: time.Now().UTC(),
	}
	s.jobs[id] = j
	return j
}

func (s *Store) CreateDone(source, mailbox string, total int) Job {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seqNo++
	id := time.Now().UTC().Format("20060102150405") + "-mig-" + source
	j := Job{
		ID:         id,
		Source:     source,
		Mailbox:    mailbox,
		Status:     "done",
		ItemsTotal: total,
		ItemsOK:    total,
		CreatedAt:  time.Now().UTC(),
	}
	s.jobs[id] = j
	return j
}

func (s *Store) NewJob(source, mailbox string, total int) Job {
	return s.CreateDone(source, mailbox, total)
}

func (s *Store) SetStatus(id, status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, ok := s.jobs[id]
	if !ok {
		return
	}
	j.Status = status
	s.jobs[id] = j
}

func (s *Store) Complete(id string, total, ok, fail int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, exists := s.jobs[id]
	if !exists {
		return
	}
	j.Status = "done"
	j.ItemsTotal = total
	j.ItemsOK = ok
	j.ItemsFail = fail
	s.jobs[id] = j
}

func (s *Store) Fail(id, errMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, exists := s.jobs[id]
	if !exists {
		return
	}
	j.Status = "failed"
	j.Error = errMsg
	s.jobs[id] = j
}

func (s *Store) MarkSeen(uid string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seen[uid] = true
}

func (s *Store) Seen(uid string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.seen[uid]
}

func (s *Store) Rerun(sourceUIDs []string) (newItems int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, uid := range sourceUIDs {
		if s.seen[uid] {
			continue
		}
		s.seen[uid] = true
		newItems++
	}
	return newItems
}

func (s *Store) Get(id string) (Job, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	j, ok := s.jobs[id]
	return j, ok
}

func (s *Store) DequeueQueued() (Job, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, j := range s.jobs {
		if j.Status == "queued" {
			j.Status = "running"
			s.jobs[id] = j
			return j, true
		}
	}
	return Job{}, false
}
