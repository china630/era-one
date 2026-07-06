// Package tenant — организации, домены и multi-tenancy (shared SaaS, ADR-0024).
package tenant

import (
	"errors"
	"strings"
	"sync"
)

var (
	ErrNotFound   = errors.New("tenant: not found")
	ErrDuplicate  = errors.New("tenant: duplicate slug or domain")
	ErrInvalidSlug = errors.New("tenant: invalid slug")
)

// Status — жизненный цикл tenant.
type Status string

const (
	StatusActive   Status = "active"
	StatusSuspended Status = "suspended"
)

// Tenant — верхний уровень изоляции (организация-заказчик).
type Tenant struct {
	ID     string
	Name   string
	Slug   string
	Status Status
}

// Org — подразделение внутри tenant (опционально).
type Org struct {
	ID       string
	TenantID string
	Name     string
}

// Domain — почтовый/SSO домен tenant.
type Domain struct {
	ID       string
	TenantID string
	FQDN     string
	Primary  bool
}

// Store — in-memory tenant registry (MVP shell).
type Store struct {
	mu      sync.RWMutex
	tenants map[string]Tenant
	orgs    map[string]Org
	domains map[string]Domain
}

// NewStore создаёт пустой реестр tenant.
func NewStore() *Store {
	return &Store{
		tenants: make(map[string]Tenant),
		orgs:    make(map[string]Org),
		domains: make(map[string]Domain),
	}
}

// PutTenant регистрирует или обновляет tenant.
func (s *Store) PutTenant(t Tenant) error {
	if t.ID == "" || t.Slug == "" {
		return ErrInvalidSlug
	}
	t.Slug = strings.ToLower(strings.TrimSpace(t.Slug))
	if t.Status == "" {
		t.Status = StatusActive
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, existing := range s.tenants {
		if existing.Slug == t.Slug && id != t.ID {
			return ErrDuplicate
		}
	}
	s.tenants[t.ID] = t
	return nil
}

// GetTenant возвращает tenant по ID.
func (s *Store) GetTenant(id string) (Tenant, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tenants[id]
	if !ok {
		return Tenant{}, ErrNotFound
	}
	return t, nil
}

// ResolveByDomain находит tenant по FQDN (для Autodiscover/SSO).
func (s *Store) ResolveByDomain(fqdn string) (Tenant, error) {
	fqdn = strings.ToLower(strings.TrimSpace(fqdn))
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, d := range s.domains {
		if strings.ToLower(d.FQDN) == fqdn {
			t, ok := s.tenants[d.TenantID]
			if !ok {
				return Tenant{}, ErrNotFound
			}
			return t, nil
		}
	}
	return Tenant{}, ErrNotFound
}

// PutDomain регистрирует домен tenant.
func (s *Store) PutDomain(d Domain) error {
	if d.TenantID == "" || d.FQDN == "" {
		return errors.New("tenant: domain requires tenant_id and fqdn")
	}
	d.FQDN = strings.ToLower(strings.TrimSpace(d.FQDN))
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tenants[d.TenantID]; !ok {
		return ErrNotFound
	}
	for id, existing := range s.domains {
		if existing.FQDN == d.FQDN && id != d.ID {
			return ErrDuplicate
		}
	}
	s.domains[d.ID] = d
	return nil
}

// PutOrg регистрирует подразделение.
func (s *Store) PutOrg(o Org) error {
	if o.TenantID == "" || o.ID == "" {
		return errors.New("tenant: org requires id and tenant_id")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tenants[o.TenantID]; !ok {
		return ErrNotFound
	}
	s.orgs[o.ID] = o
	return nil
}
