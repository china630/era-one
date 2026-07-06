package store

import (
	"sync"
	"time"
)

// Installation — зарегистрированная инсталляция заказчика.
type Installation struct {
	DeploymentID string    `json:"deployment_id"`
	TenantID     string    `json:"tenant_id"`
	LicenseID    string    `json:"license_id"`
	Customer     string    `json:"customer"`
	LastHealthAt time.Time `json:"last_health_at,omitempty"`
	HealthLevel  string    `json:"health_level,omitempty"`
	AgentCount   int       `json:"agent_count,omitempty"`
	LeaseStatus  string    `json:"lease_status,omitempty"`
}

// Store — in-memory реестр Portal v0.
type Store struct {
	mu            sync.RWMutex
	installations map[string]*Installation
	healthLog     map[string][]map[string]any
	crlToken      string
}

func New() *Store {
	return &Store{
		installations: make(map[string]*Installation),
		healthLog:     make(map[string][]map[string]any),
	}
}

func (s *Store) UpsertInstallation(i *Installation) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.installations[i.DeploymentID] = i
}

func (s *Store) GetInstallation(id string) (*Installation, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	i, ok := s.installations[id]
	return i, ok
}

func (s *Store) ListInstallations() []*Installation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Installation, 0, len(s.installations))
	for _, i := range s.installations {
		out = append(out, i)
	}
	return out
}

func (s *Store) RecordHealth(deploymentID string, payload map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.healthLog[deploymentID] = append(s.healthLog[deploymentID], payload)
	if len(s.healthLog[deploymentID]) > 100 {
		s.healthLog[deploymentID] = s.healthLog[deploymentID][len(s.healthLog[deploymentID])-100:]
	}
	if inst, ok := s.installations[deploymentID]; ok {
		inst.LastHealthAt = time.Now().UTC()
		if v, ok := payload["level"].(string); ok {
			inst.HealthLevel = v
		}
		if v, ok := payload["agent_count"].(float64); ok {
			inst.AgentCount = int(v)
		}
		if v, ok := payload["lease_status"].(string); ok {
			inst.LeaseStatus = v
		}
	}
}

func (s *Store) SetCRL(token string) {
	s.mu.Lock()
	s.crlToken = token
	s.mu.Unlock()
}

func (s *Store) CRL() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.crlToken
}
