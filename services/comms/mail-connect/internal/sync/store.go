package sync

import (
	"strconv"
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
	ItemsTotal int       `json:"items_total"`
	ItemsOK    int       `json:"items_ok"`
	ItemsFail  int       `json:"items_fail"`
	CreatedAt  time.Time `json:"created_at"`
}

type Store struct {
	mu       sync.RWMutex
	mailbox  map[string]Mailbox
	jobs     map[string]Job
	sequence int
}

func NewStore() *Store {
	return &Store{
		mailbox: make(map[string]Mailbox),
		jobs:    make(map[string]Job),
	}
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

func (s *Store) StartSync(tenantID, mailbox string) Job {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sequence++
	id := "sync-" + time.Now().UTC().Format("20060102150405") + "-" + strconv.Itoa(s.sequence)
	j := Job{
		ID:         id,
		Mailbox:    mailbox,
		TenantID:   tenantID,
		Status:     "done",
		ItemsTotal: 12,
		ItemsOK:    12,
		ItemsFail:  0,
		CreatedAt:  time.Now().UTC(),
	}
	s.jobs[j.ID] = j
	return j
}

func (s *Store) GetJob(id string) (Job, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	j, ok := s.jobs[id]
	return j, ok
}
