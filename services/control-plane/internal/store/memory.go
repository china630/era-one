package store

import (
	"sync"
	"time"
)

type memoryStore struct {
	mu          sync.RWMutex
	assets      map[string]*Asset
	software    map[string][]*AssetSoftware
	contracts   map[string]*Contract
	licenses    map[string]*SoftwareLicense
	cases       map[string]*Case
	notes       map[string][]*CaseNote
	timeline    map[string][]*TimelineEvent
	audit       []*AuditEntry
	policy      PolicyBundle
	hybrid      hybridFields
	enforcement EnforcementPolicy
	enforcementPrev EnforcementPolicy
	enforcementHistory []EnforcementPolicyRevision
	escrows     map[string]*BitlockerEscrow
	deployJobs  []*DeployJob
	patchJobs   []*PatchJob
	tenantFilter string
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		assets:    make(map[string]*Asset),
		software:  make(map[string][]*AssetSoftware),
		contracts: make(map[string]*Contract),
		licenses:  make(map[string]*SoftwareLicense),
		cases:    make(map[string]*Case),
		notes:    make(map[string][]*CaseNote),
		timeline: make(map[string][]*TimelineEvent),
		policy:   DefaultPolicy(),
		hybrid:   hybridFields{policy: DefaultHybridPolicy()},
		enforcement: DefaultEnforcementPolicy(),
		escrows:  make(map[string]*BitlockerEscrow),
	}
}

func (m *memoryStore) Close() error { return nil }

func (m *memoryStore) UpsertAsset(a *Asset) {
	m.UpsertAssetFull(a)
}

func (m *memoryStore) ListAssets() []*Asset {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*Asset, 0, len(m.assets))
	for _, a := range m.assets {
		if m.tenantFilter != "" && a.TenantID != m.tenantFilter {
			continue
		}
		out = append(out, a)
	}
	return out
}

func (m *memoryStore) AssetCoverage() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.assets) == 0 {
		return 0
	}
	active := 0
	cutoff := nowUTC().Add(-15 * time.Minute)
	for _, a := range m.assets {
		if a.LastSeen.After(cutoff) {
			active++
		}
	}
	return float64(active) / float64(len(m.assets))
}

func (m *memoryStore) CreateCase(c *Case) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := nowUTC()
	c.CreatedAt = now
	c.UpdatedAt = now
	if c.Status == "" {
		c.Status = "new"
	}
	m.cases[c.ID] = c
	ev := newTimeline(c.ID, "created", "system", "case opened")
	m.timeline[c.ID] = append(m.timeline[c.ID], ev)
}

func (m *memoryStore) GetCase(id string) (*Case, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.cases[id]
	return c, ok
}

func (m *memoryStore) UpdateCase(id string, fn func(*Case)) (*Case, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.cases[id]
	if !ok {
		return nil, false
	}
	fn(c)
	c.UpdatedAt = nowUTC()
	return c, true
}

func (m *memoryStore) ListCases() []*Case {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*Case, 0, len(m.cases))
	for _, c := range m.cases {
		if m.tenantFilter != "" && c.TenantID != m.tenantFilter {
			continue
		}
		out = append(out, c)
	}
	return out
}

func (m *memoryStore) Policy() PolicyBundle {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.policy
}

func (m *memoryStore) SetPolicy(p PolicyBundle) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.policy = p
}
