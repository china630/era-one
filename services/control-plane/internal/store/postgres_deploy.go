package store

import (
	"time"

	"github.com/google/uuid"
)

func (s *postgresStore) deployMigrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS deploy_jobs (
  id TEXT PRIMARY KEY,
  tenant_id TEXT,
  node_id TEXT NOT NULL,
  package_ref TEXT NOT NULL,
  ota_token TEXT,
  reboot BOOLEAN DEFAULT FALSE,
  status TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
)`,
		`CREATE TABLE IF NOT EXISTS patch_jobs (
  id TEXT PRIMARY KEY,
  tenant_id TEXT,
  node_id TEXT NOT NULL,
  cve_id TEXT NOT NULL,
  product TEXT,
  package_ref TEXT NOT NULL,
  status TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL
)`,
	}
	for _, q := range stmts {
		if _, err := s.db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

func (s *postgresStore) CreateDeployJob(j *DeployJob) {
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
	_, _ = s.db.Exec(
		`INSERT INTO deploy_jobs (id, tenant_id, node_id, package_ref, ota_token, reboot, status, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		j.ID, j.TenantID, j.NodeID, j.PackageRef, j.OTAToken, j.Reboot, string(j.Status), j.CreatedAt, j.UpdatedAt,
	)
}

func (s *postgresStore) ListDeployJobs() []*DeployJob {
	rows, err := s.db.Query(`SELECT id, tenant_id, node_id, package_ref, ota_token, reboot, status, created_at, updated_at FROM deploy_jobs`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []*DeployJob
	for rows.Next() {
		var j DeployJob
		var st string
		if err := rows.Scan(&j.ID, &j.TenantID, &j.NodeID, &j.PackageRef, &j.OTAToken, &j.Reboot, &st, &j.CreatedAt, &j.UpdatedAt); err != nil {
			continue
		}
		j.Status = RolloutStatus(st)
		out = append(out, &j)
	}
	return out
}

func (s *postgresStore) UpdateDeployJob(id string, status RolloutStatus) (*DeployJob, bool) {
	now := time.Now().UTC()
	res, err := s.db.Exec(`UPDATE deploy_jobs SET status=$1, updated_at=$2 WHERE id=$3`, string(status), now, id)
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

func (s *postgresStore) CreatePatchJob(j *PatchJob) {
	if j == nil {
		return
	}
	if j.ID == "" {
		j.ID = uuid.NewString()
	}
	if j.CreatedAt.IsZero() {
		j.CreatedAt = time.Now().UTC()
	}
	if j.Status == "" {
		j.Status = RolloutPending
	}
	_, _ = s.db.Exec(
		`INSERT INTO patch_jobs (id, tenant_id, node_id, cve_id, product, package_ref, status, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		j.ID, j.TenantID, j.NodeID, j.CVEID, j.Product, j.PackageRef, string(j.Status), j.CreatedAt,
	)
}

func (s *postgresStore) ListPatchJobs() []*PatchJob {
	rows, err := s.db.Query(`SELECT id, tenant_id, node_id, cve_id, product, package_ref, status, created_at FROM patch_jobs`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []*PatchJob
	for rows.Next() {
		var j PatchJob
		var st string
		if err := rows.Scan(&j.ID, &j.TenantID, &j.NodeID, &j.CVEID, &j.Product, &j.PackageRef, &st, &j.CreatedAt); err != nil {
			continue
		}
		j.Status = RolloutStatus(st)
		out = append(out, &j)
	}
	return out
}

func (s *postgresStore) PlanPatches(catalog []PatchCatalogEntry) []PatchPlanRow {
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
