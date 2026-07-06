package store

import (
	"time"

	"github.com/google/uuid"
)

type hybridFields struct {
	policy     HybridPolicy
	runtime    HybridRuntime
	egress     []*EgressAuditEntry
	leaseToken string
	leaseRenew time.Time
}

func (m *memoryStore) GetHybridPolicy() HybridPolicy {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.hybrid.policy
}

func (m *memoryStore) SetHybridPolicy(p HybridPolicy) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hybrid.policy = p
}

func (m *memoryStore) GetHybridRuntime() HybridRuntime {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.hybrid.runtime
}

func (m *memoryStore) SetHybridRuntime(r HybridRuntime) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hybrid.runtime = r
}

func (m *memoryStore) RecordEgressAudit(e *EgressAuditEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	if e.At.IsZero() {
		e.At = nowUTC()
	}
	m.hybrid.egress = append(m.hybrid.egress, e)
	if len(m.hybrid.egress) > 500 {
		m.hybrid.egress = m.hybrid.egress[len(m.hybrid.egress)-500:]
	}
}

func (m *memoryStore) ListEgressAudit(limit int) []*EgressAuditEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if limit <= 0 || limit > len(m.hybrid.egress) {
		limit = len(m.hybrid.egress)
	}
	start := len(m.hybrid.egress) - limit
	if start < 0 {
		start = 0
	}
	out := make([]*EgressAuditEntry, limit)
	copy(out, m.hybrid.egress[start:])
	return out
}

func (m *memoryStore) GetLeaseCache() (string, time.Time) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.hybrid.leaseToken, m.hybrid.leaseRenew
}

func (m *memoryStore) SetLeaseCache(token string, lastRenew time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hybrid.leaseToken = token
	m.hybrid.leaseRenew = lastRenew
	if lastRenew.IsZero() {
		m.hybrid.leaseRenew = time.Now().UTC()
	}
}
