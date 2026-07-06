package store

import "fmt"

func (m *memoryStore) GetEnforcementPolicy() EnforcementPolicy {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.enforcement.Version == "" {
		return DefaultEnforcementPolicy()
	}
	return m.enforcement
}

func (m *memoryStore) SetEnforcementPolicy(p EnforcementPolicy, actor, detail string) error {
	if p.Version == "" {
		return fmt.Errorf("policy version required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.enforcement.Version != "" {
		m.enforcementPrev = m.enforcement
	}
	m.enforcement = p
	m.enforcementHistory = append([]EnforcementPolicyRevision{{
		Version:   p.Version,
		Mode:      p.Mode,
		Actor:     actor,
		Detail:    detail,
		CreatedAt: nowUTC(),
	}}, m.enforcementHistory...)
	if len(m.enforcementHistory) > 50 {
		m.enforcementHistory = m.enforcementHistory[:50]
	}
	return nil
}

func (m *memoryStore) RollbackEnforcementPolicy(actor string) (EnforcementPolicy, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.enforcementPrev.Version == "" {
		return m.enforcement, false
	}
	cur := m.enforcement
	m.enforcement = m.enforcementPrev
	m.enforcementPrev = EnforcementPolicy{}
	m.enforcementHistory = append([]EnforcementPolicyRevision{{
		Version:   m.enforcement.Version,
		Mode:      m.enforcement.Mode,
		Actor:     actor,
		Detail:    "rollback from " + cur.Version,
		CreatedAt: nowUTC(),
	}}, m.enforcementHistory...)
	return m.enforcement, true
}

func (m *memoryStore) ListEnforcementHistory(limit int) []EnforcementPolicyRevision {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if limit <= 0 || limit > len(m.enforcementHistory) {
		limit = len(m.enforcementHistory)
	}
	out := make([]EnforcementPolicyRevision, limit)
	copy(out, m.enforcementHistory[:limit])
	return out
}

func (m *memoryStore) UpsertBitlockerEscrow(e *BitlockerEscrow) {
	if e == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.escrows == nil {
		m.escrows = make(map[string]*BitlockerEscrow)
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = nowUTC()
	}
	key := e.NodeID + "|" + e.VolumeID
	m.escrows[key] = e
}

func (m *memoryStore) GetBitlockerEscrow(nodeID, volumeID string) (*BitlockerEscrow, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	e, ok := m.escrows[nodeID+"|"+volumeID]
	return e, ok
}

func (m *memoryStore) ListBitlockerEscrows(nodeID string) []BitlockerEscrowPublic {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []BitlockerEscrowPublic
	for _, e := range m.escrows {
		if e == nil {
			continue
		}
		if nodeID != "" && e.NodeID != nodeID {
			continue
		}
		if m.tenantFilter != "" && e.TenantID != m.tenantFilter {
			continue
		}
		out = append(out, e.Public())
	}
	return out
}
