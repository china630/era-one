package store

import (
	"time"

	"github.com/google/uuid"
)

func (s *sqliteStore) deployMigrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS deploy_jobs (
  id TEXT PRIMARY KEY,
  tenant_id TEXT,
  node_id TEXT NOT NULL,
  package_ref TEXT NOT NULL,
  ota_token TEXT,
  reboot INTEGER DEFAULT 0,
  status TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
)`,
		`CREATE TABLE IF NOT EXISTS patch_jobs (
  id TEXT PRIMARY KEY,
  tenant_id TEXT,
  node_id TEXT NOT NULL,
  cve_id TEXT NOT NULL,
  product TEXT,
  package_ref TEXT NOT NULL,
  status TEXT NOT NULL,
  created_at TEXT NOT NULL
)`,
	}
	for _, q := range stmts {
		if _, err := s.db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

func (s *sqliteStore) CreateDeployJob(j *DeployJob) {
	if j == nil {
		return
	}
	if j.ID == "" {
		j.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	if j.CreatedAt.IsZero() {
		j.CreatedAt = now
	}
	j.UpdatedAt = now
	if j.Status == "" {
		j.Status = RolloutPending
	}
	reboot := 0
	if j.Reboot {
		reboot = 1
	}
	_, _ = s.db.Exec(
		`INSERT INTO deploy_jobs (id, tenant_id, node_id, package_ref, ota_token, reboot, status, created_at, updated_at)
		 VALUES (?,?,?,?,?,?,?,?,?)`,
		j.ID, j.TenantID, j.NodeID, j.PackageRef, j.OTAToken, reboot, string(j.Status),
		j.CreatedAt.Format(time.RFC3339Nano), j.UpdatedAt.Format(time.RFC3339Nano),
	)
}

func (s *sqliteStore) ListDeployJobs() []*DeployJob {
	rows, err := s.db.Query(`SELECT id, tenant_id, node_id, package_ref, ota_token, reboot, status, created_at, updated_at FROM deploy_jobs`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []*DeployJob
	for rows.Next() {
		var j DeployJob
		var st string
		var reboot int
		var cts, uts string
		if err := rows.Scan(&j.ID, &j.TenantID, &j.NodeID, &j.PackageRef, &j.OTAToken, &reboot, &st, &cts, &uts); err != nil {
			continue
		}
		j.Reboot = reboot == 1
		j.Status = RolloutStatus(st)
		j.CreatedAt, _ = time.Parse(time.RFC3339Nano, cts)
		j.UpdatedAt, _ = time.Parse(time.RFC3339Nano, uts)
		out = append(out, &j)
	}
	return out
}

func (s *sqliteStore) UpdateDeployJob(id string, status RolloutStatus) (*DeployJob, bool) {
	uts := time.Now().UTC().Format(time.RFC3339Nano)
	res, err := s.db.Exec(`UPDATE deploy_jobs SET status=?, updated_at=? WHERE id=?`, string(status), uts, id)
	if err != nil {
		return nil, false
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, false
	}
	for _, j := range s.ListDeployJobs() {
		if j.ID == id {
			return j, true
		}
	}
	return nil, false
}

func (s *sqliteStore) CreatePatchJob(j *PatchJob) {
	if j == nil {
		return
	}
	if j.ID == "" {
		j.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	if j.CreatedAt.IsZero() {
		j.CreatedAt = now
	}
	if j.Status == "" {
		j.Status = RolloutPending
	}
	_, _ = s.db.Exec(
		`INSERT INTO patch_jobs (id, tenant_id, node_id, cve_id, product, package_ref, status, created_at) VALUES (?,?,?,?,?,?,?,?)`,
		j.ID, j.TenantID, j.NodeID, j.CVEID, j.Product, j.PackageRef, string(j.Status), j.CreatedAt.Format(time.RFC3339Nano),
	)
}

func (s *sqliteStore) ListPatchJobs() []*PatchJob {
	rows, err := s.db.Query(`SELECT id, tenant_id, node_id, cve_id, product, package_ref, status, created_at FROM patch_jobs`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []*PatchJob
	for rows.Next() {
		var j PatchJob
		var st, ts string
		if err := rows.Scan(&j.ID, &j.TenantID, &j.NodeID, &j.CVEID, &j.Product, &j.PackageRef, &st, &ts); err != nil {
			continue
		}
		j.Status = RolloutStatus(st)
		j.CreatedAt, _ = time.Parse(time.RFC3339Nano, ts)
		out = append(out, &j)
	}
	return out
}

func (s *sqliteStore) PlanPatches(catalog []PatchCatalogEntry) []PatchPlanRow {
	sw := s.ListAllAssetSoftware()
	m := make(map[string][]*AssetSoftware)
	for _, row := range sw {
		if row == nil {
			continue
		}
		m[row.NodeID] = append(m[row.NodeID], row)
	}
	return planPatchesFromSoftware(m, catalog)
}
