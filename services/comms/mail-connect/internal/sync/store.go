package sync

import (
	"sync"
	"time"
)

type Mailbox struct {
	TenantID    string `json:"tenant_id"`
	Email       string `json:"email"`
	Provider    string `json:"provider"`
	Address     string `json:"address"`
	Username    string `json:"username"`
	PasswordRef string `json:"password_ref"`
}

type Job struct {
	ID         string    `json:"id"`
	Mailbox    string    `json:"mailbox"`
	TenantID   string    `json:"tenant_id"`
	Status     string    `json:"status"`
	Mode       string    `json:"mode"` // live | stub (G0-7 honesty)
	ItemsTotal int       `json:"items_total"`
	ItemsOK    int       `json:"items_ok"`
	ItemsFail  int       `json:"items_fail"`
	Error      string    `json:"error,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type Store struct {
	mu       sync.RWMutex
	mailbox  map[string]Mailbox
	jobs     map[string]Job
	cursors  map[string]uint32
	sequence int
}

func NewStore() *Store {
	return &Store{
		mailbox: make(map[string]Mailbox),
		jobs:    make(map[string]Job),
		cursors: make(map[string]uint32),
	}
}

func (s *Store) Cursor(tenantID, mailbox, folder string) uint32 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cursors[key(tenantID, mailbox)+":"+folder]
}

func key(tenantID, email string) string {
	return tenantID + "/" + email
}

func (s *Store) PutMailbox(m Mailbox) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mailbox[key(m.TenantID, m.Email)] = m
}

func (s *Store) GetMailbox(tenantID, email string) (Mailbox, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.mailbox[key(tenantID, email)]
	return m, ok
}

func (s *Store) GetJob(id string) (Job, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	j, ok := s.jobs[id]
	return j, ok
}
