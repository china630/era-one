package store

import (
	_ "embed"
	"encoding/json"
)

//go:embed testdata/patch_catalog.json
var patchCatalogJSON []byte

// DefaultPatchCatalog — dev CVE→package mapping (air-gap MinIO refs).
func DefaultPatchCatalog() []PatchCatalogEntry {
	var rows []PatchCatalogEntry
	if err := json.Unmarshal(patchCatalogJSON, &rows); err != nil {
		return nil
	}
	return rows
}

func (m *memoryStore) CreateDeployJob(j *DeployJob) {
	if j == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	now := nowUTC()
	if j.CreatedAt.IsZero() {
		j.CreatedAt = now
	}
	j.UpdatedAt = now
	if j.Status == "" {
		j.Status = RolloutPending
	}
	m.deployJobs = append(m.deployJobs, j)
}

func (m *memoryStore) ListDeployJobs() []*DeployJob {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]*DeployJob(nil), m.deployJobs...)
}

func (m *memoryStore) UpdateDeployJob(id string, status RolloutStatus) (*DeployJob, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, j := range m.deployJobs {
		if j.ID == id {
			j.Status = status
			j.UpdatedAt = nowUTC()
			return j, true
		}
	}
	return nil, false
}

func (m *memoryStore) CreatePatchJob(j *PatchJob) {
	if j == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if j.CreatedAt.IsZero() {
		j.CreatedAt = nowUTC()
	}
	if j.Status == "" {
		j.Status = RolloutPending
	}
	m.patchJobs = append(m.patchJobs, j)
}

func (m *memoryStore) ListPatchJobs() []*PatchJob {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]*PatchJob(nil), m.patchJobs...)
}

func (m *memoryStore) PlanPatches(catalog []PatchCatalogEntry) []PatchPlanRow {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return planPatchesFromSoftware(m.software, catalog)
}
