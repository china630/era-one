// Package hold — arbitration-like store + TTL (AC-MM-2,5).
package hold

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Status hold lifecycle.
type Status string

const (
	StatusPending  Status = "pending"
	StatusApproved Status = "approved"
	StatusRejected Status = "rejected"
	StatusExpired  Status = "expired"
)

// Record — письмо на визировании.
type Record struct {
	ID          string
	Status      Status
	RuleID      string
	Sender      string
	Recipients  []string
	Subject     string
	Moderators  []string
	Raw         []byte
	Comment     string
	ExpiresAt   time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ConsumedBy  string // moderator who acted (any-of)
}

// Store — in-memory hold.
type Store struct {
	mu   sync.Mutex
	byID map[string]*Record
	now  func() time.Time
}

// Repository — hold persistence (memory или PG).
type Repository interface {
	Put(r Record) (Record, error)
	Get(id string) (Record, bool)
	Approve(id, moderator string) (Record, error)
	Reject(id, moderator, comment string) (Record, error)
	ExpirePending(autoApprove bool) []Record
}

// ListPending — опционально для Admin UI.
type Lister interface {
	ListPending() []Record
}

func NewStore() *Store {
	return &Store{
		byID: make(map[string]*Record),
		now:  time.Now,
	}
}

// Put создаёт pending hold.
func (s *Store) Put(r Record) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r.ID == "" {
		id, err := newID()
		if err != nil {
			return Record{}, err
		}
		r.ID = id
	}
	now := s.now()
	r.Status = StatusPending
	r.CreatedAt = now
	r.UpdatedAt = now
	if r.ExpiresAt.IsZero() {
		r.ExpiresAt = now.Add(72 * time.Hour)
	}
	cp := r
	cp.Raw = append([]byte(nil), r.Raw...)
	s.byID[r.ID] = &cp
	return cp, nil
}

func (s *Store) Get(id string) (Record, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.byID[id]
	if !ok {
		return Record{}, false
	}
	out := *r
	out.Raw = append([]byte(nil), r.Raw...)
	return out, true
}

// Approve — any-of: первый побеждает.
func (s *Store) Approve(id, moderator string) (Record, error) {
	return s.act(id, moderator, StatusApproved, "")
}

// Reject требует comment.
func (s *Store) Reject(id, moderator, comment string) (Record, error) {
	if comment == "" {
		return Record{}, fmt.Errorf("reject comment required")
	}
	return s.act(id, moderator, StatusRejected, comment)
}

func (s *Store) act(id, moderator string, st Status, comment string) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.byID[id]
	if !ok {
		return Record{}, fmt.Errorf("hold %s not found", id)
	}
	if r.Status != StatusPending {
		return Record{}, fmt.Errorf("hold %s status %s", id, r.Status)
	}
	if !isModerator(r.Moderators, moderator) && moderator != "admin" {
		return Record{}, fmt.Errorf("moderator %s not allowed", moderator)
	}
	r.Status = st
	r.Comment = comment
	r.ConsumedBy = moderator
	r.UpdatedAt = s.now()
	out := *r
	out.Raw = append([]byte(nil), r.Raw...)
	return out, nil
}

// ExpirePending помечает просроченные (default auto-reject).
func (s *Store) ExpirePending(autoApprove bool) []Record {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	var out []Record
	for _, r := range s.byID {
		if r.Status != StatusPending || !now.After(r.ExpiresAt) {
			continue
		}
		if autoApprove {
			r.Status = StatusApproved
		} else {
			r.Status = StatusExpired
			r.Comment = "ttl expired"
		}
		r.UpdatedAt = now
		cp := *r
		cp.Raw = append([]byte(nil), r.Raw...)
		out = append(out, cp)
	}
	return out
}

func isModerator(list []string, who string) bool {
	for _, m := range list {
		if strings.EqualFold(m, who) {
			return true
		}
	}
	return false
}

func newID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

// SetClockForTest — только тесты.
func (s *Store) SetClockForTest(now func() time.Time) {
	s.now = now
}

// ListPending возвращает pending holds.
func (s *Store) ListPending() []Record {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []Record
	for _, r := range s.byID {
		if r.Status == StatusPending {
			cp := *r
			cp.Raw = append([]byte(nil), r.Raw...)
			out = append(out, cp)
		}
	}
	return out
}
