package store

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// Suppression — FP suppression rule (ADR-0022 DC-7).
type Suppression struct {
	ID        string     `json:"id"`
	TenantID  string     `json:"tenant_id,omitempty"`
	RuleID    string     `json:"rule_id"`
	NodeID    string     `json:"node_id,omitempty"` // empty = rule-wide
	Reason    string     `json:"reason,omitempty"`
	CreatedBy string     `json:"created_by,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// SuppressionMem — in-memory suppressions (MVP; all store drivers share this for DC-02).
type SuppressionMem struct {
	mu   sync.RWMutex
	byID map[string]*Suppression
}

func NewSuppressionMem() *SuppressionMem {
	return &SuppressionMem{byID: map[string]*Suppression{}}
}

func (s *SuppressionMem) Create(sp *Suppression) *Suppression {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sp.ID == "" {
		sp.ID = uuid.NewString()
	}
	if sp.CreatedAt.IsZero() {
		sp.CreatedAt = time.Now().UTC()
	}
	cp := *sp
	s.byID[sp.ID] = &cp
	return &cp
}

func (s *SuppressionMem) List() []*Suppression {
	s.mu.RLock()
	defer s.mu.RUnlock()
	now := time.Now().UTC()
	out := make([]*Suppression, 0, len(s.byID))
	for _, sp := range s.byID {
		if sp.ExpiresAt != nil && sp.ExpiresAt.Before(now) {
			continue
		}
		cp := *sp
		out = append(out, &cp)
	}
	return out
}

func (s *SuppressionMem) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.byID[id]; !ok {
		return false
	}
	delete(s.byID, id)
	return true
}

// Matches reports whether (ruleID, nodeID) is suppressed.
func (s *SuppressionMem) Matches(tenantID, ruleID, nodeID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	now := time.Now().UTC()
	for _, sp := range s.byID {
		if sp.ExpiresAt != nil && sp.ExpiresAt.Before(now) {
			continue
		}
		if sp.TenantID != "" && tenantID != "" && sp.TenantID != tenantID {
			continue
		}
		if sp.RuleID != ruleID {
			continue
		}
		if sp.NodeID == "" || sp.NodeID == nodeID {
			return true
		}
	}
	return false
}
