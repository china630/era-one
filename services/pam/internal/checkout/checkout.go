// Package checkout — brokering креденшелов (RBAC + approval + TTL).
package checkout

import (
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusPending   Status = "pending_approval"
	StatusApproved  Status = "approved"
	StatusDenied    Status = "denied"
	StatusExpired   Status = "expired"
	StatusConsumed  Status = "consumed"
	StatusRevoked   Status = "revoked"
)

type Request struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	SecretID   string    `json:"secret_id"`
	Requester  string    `json:"requester"`
	Approver   string    `json:"approver,omitempty"`
	Status     Status    `json:"status"`
	TTLMinutes int       `json:"ttl_minutes"`
	ExpiresAt  time.Time `json:"expires_at"`
	CreatedAt  time.Time `json:"created_at"`
}

type Store struct {
	mu    sync.Mutex
	items map[string]*Request
}

func NewStore() *Store {
	return &Store{items: make(map[string]*Request)}
}

func (s *Store) Create(tenantID, secretID, requester string, ttlMinutes int, autoApprove bool) (*Request, error) {
	if secretID == "" || requester == "" {
		return nil, errors.New("secret_id and requester required")
	}
	if ttlMinutes <= 0 {
		ttlMinutes = 60
	}
	now := time.Now().UTC()
	r := &Request{
		ID: uuid.NewString(), TenantID: tenantID, SecretID: secretID,
		Requester: requester, TTLMinutes: ttlMinutes,
		ExpiresAt: now.Add(time.Duration(ttlMinutes) * time.Minute),
		CreatedAt: now,
		Status:    StatusPending,
	}
	if autoApprove {
		r.Status = StatusApproved
	}
	s.mu.Lock()
	s.items[r.ID] = r
	s.mu.Unlock()
	return r, nil
}

func (s *Store) Get(id string) (*Request, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.items[id]
	if !ok {
		return nil, false
	}
	if r.Status == StatusApproved && time.Now().UTC().After(r.ExpiresAt) {
		r.Status = StatusExpired
	}
	return r, true
}

func (s *Store) Approve(id, approver string) (*Request, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.items[id]
	if !ok || r.Status != StatusPending {
		return nil, false
	}
	r.Status = StatusApproved
	r.Approver = approver
	return r, true
}

func (s *Store) Deny(id, approver string) (*Request, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.items[id]
	if !ok || r.Status != StatusPending {
		return nil, false
	}
	r.Status = StatusDenied
	r.Approver = approver
	return r, true
}

func (s *Store) Consume(id string) (*Request, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.items[id]
	if !ok || r.Status != StatusApproved {
		return nil, false
	}
	if time.Now().UTC().After(r.ExpiresAt) {
		r.Status = StatusExpired
		return nil, false
	}
	r.Status = StatusConsumed
	return r, true
}

func (s *Store) Revoke(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.items[id]
	if !ok {
		return false
	}
	r.Status = StatusRevoked
	return true
}

func (s *Store) List() []*Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*Request, 0, len(s.items))
	for _, r := range s.items {
		out = append(out, r)
	}
	return out
}

// PolicyAllow — RBAC: admin auto-approve; requester needs approval.
func PolicyAllow(role string) (autoApprove bool, allowed bool) {
	switch role {
	case "admin", "pam-admin":
		return true, true
	case "analyst", "operator":
		return false, true
	default:
		return false, false
	}
}
