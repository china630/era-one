package store

import (
	"github.com/google/uuid"
)

func (m *memoryStore) GetAsset(nodeID string) (*Asset, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	a, ok := m.assets[nodeID]
	return a, ok
}

func (m *memoryStore) FindAssetByAgentID(agentID string) (*Asset, bool) {
	if agentID == "" {
		return nil, false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, a := range m.assets {
		if a.AgentID == agentID {
			return a, true
		}
	}
	return nil, false
}

func (m *memoryStore) FindAssetBySerial(serial string) (*Asset, bool) {
	if serial == "" {
		return nil, false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, a := range m.assets {
		if a.SerialNumber == serial || a.BoardSerial == serial {
			return a, true
		}
	}
	return nil, false
}

func (m *memoryStore) UpsertAssetFull(a *Asset) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a.LastSeen = nowUTC()
	if a.InventoryUpdatedAt.IsZero() {
		a.InventoryUpdatedAt = a.LastSeen
	}
	m.assets[a.NodeID] = a
}

func (m *memoryStore) ReplaceAssetSoftware(nodeID, tenantID string, items []*AssetSoftware) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := nowUTC()
	filtered := make([]*AssetSoftware, 0, len(items))
	for _, it := range items {
		if it == nil {
			continue
		}
		it.NodeID = nodeID
		it.TenantID = tenantID
		if it.FirstSeen.IsZero() {
			it.FirstSeen = now
		}
		it.LastSeen = now
		filtered = append(filtered, it)
	}
	m.software[nodeID] = filtered
}

func (m *memoryStore) ListAssetSoftware(nodeID string) []*AssetSoftware {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]*AssetSoftware(nil), m.software[nodeID]...)
}

func (m *memoryStore) ListAllAssetSoftware() []*AssetSoftware {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*AssetSoftware
	for _, list := range m.software {
		out = append(out, list...)
	}
	return out
}

func (m *memoryStore) UpsertContract(c *Contract) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	m.contracts[c.ID] = c
}

func (m *memoryStore) ListContracts() []*Contract {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*Contract, 0, len(m.contracts))
	for _, c := range m.contracts {
		if m.tenantFilter != "" && c.TenantID != m.tenantFilter {
			continue
		}
		out = append(out, c)
	}
	return out
}

func (m *memoryStore) UpsertSoftwareLicense(l *SoftwareLicense) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if l.ID == "" {
		l.ID = uuid.NewString()
	}
	m.licenses[l.ID] = l
}

func (m *memoryStore) ListSoftwareLicenses() []*SoftwareLicense {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*SoftwareLicense, 0, len(m.licenses))
	for _, l := range m.licenses {
		if m.tenantFilter != "" && l.TenantID != m.tenantFilter {
			continue
		}
		out = append(out, l)
	}
	return out
}

func (m *memoryStore) ReconcileSoftwareLicenses() []ReconcileRow {
	return reconcileInstalledEntitled(m.ListAllAssetSoftware(), m.ListSoftwareLicenses())
}

func reconcileRow(product string, installed, entitled int) ReconcileRow {
	delta := installed - entitled
	compliance := "ok"
	switch {
	case delta > 0:
		compliance = "over"
	case delta < 0:
		compliance = "under"
	}
	return ReconcileRow{
		Product: product, Installed: installed, Entitled: entitled,
		Delta: delta, Compliance: compliance,
	}
}
