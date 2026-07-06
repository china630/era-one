// Package identity — пользователи, группы и RBAC (shared SaaS, ADR-0024).
package identity

import (
	"errors"
	"sync"
)

var (
	ErrNotFound      = errors.New("identity: not found")
	ErrDuplicate     = errors.New("identity: duplicate")
	ErrTenantMissing = errors.New("identity: tenant required")
)

// User — учётная запись в рамках tenant.
type User struct {
	ID          string
	TenantID    string
	Email       string
	DisplayName string
	Active      bool
	RoleIDs     []string
}

// Group — группа пользователей tenant.
type Group struct {
	ID       string
	TenantID string
	Name     string
	UserIDs  []string
}

// Role — именованный набор разрешений (RBAC).
type Role struct {
	ID          string
	TenantID    string
	Name        string
	Permissions []string
}

// Store — in-memory хранилище identity (MVP shell; Postgres — позже).
type Store struct {
	mu     sync.RWMutex
	users  map[string]User
	groups map[string]Group
	roles  map[string]Role
}

// NewStore создаёт пустое хранилище.
func NewStore() *Store {
	return &Store{
		users:  make(map[string]User),
		groups: make(map[string]Group),
		roles:  make(map[string]Role),
	}
}

// PutUser создаёт или обновляет пользователя.
func (s *Store) PutUser(u User) error {
	if u.TenantID == "" {
		return ErrTenantMissing
	}
	if u.ID == "" {
		return errors.New("identity: user id required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.users[u.ID]; ok && existing.TenantID != u.TenantID {
		return ErrDuplicate
	}
	s.users[u.ID] = u
	return nil
}

// GetUser возвращает пользователя по ID.
func (s *Store) GetUser(id string) (User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[id]
	if !ok {
		return User{}, ErrNotFound
	}
	return u, nil
}

// ListUsersByTenant возвращает пользователей tenant.
func (s *Store) ListUsersByTenant(tenantID string) []User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]User, 0)
	for _, u := range s.users {
		if u.TenantID == tenantID {
			out = append(out, u)
		}
	}
	return out
}

// PutRole создаёт или обновляет роль.
func (s *Store) PutRole(r Role) error {
	if r.TenantID == "" {
		return ErrTenantMissing
	}
	if r.ID == "" {
		return errors.New("identity: role id required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.roles[r.ID] = r
	return nil
}

// PermissionsForUser собирает разрешения пользователя по ролям tenant.
func (s *Store) PermissionsForUser(userID string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[userID]
	if !ok {
		return nil, ErrNotFound
	}
	seen := make(map[string]struct{})
	for _, rid := range u.RoleIDs {
		r, ok := s.roles[rid]
		if !ok || r.TenantID != u.TenantID {
			continue
		}
		for _, p := range r.Permissions {
			seen[p] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for p := range seen {
		out = append(out, p)
	}
	return out, nil
}
