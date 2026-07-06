package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

func (s *postgresStore) enforcementMigrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS enforcement_policy (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  version TEXT NOT NULL,
  mode TEXT NOT NULL,
  policy_json TEXT NOT NULL,
  prev_json TEXT
)`,
		`CREATE TABLE IF NOT EXISTS enforcement_history (
  id SERIAL PRIMARY KEY,
  version TEXT NOT NULL,
  mode TEXT NOT NULL,
  actor TEXT,
  detail TEXT,
  created_at TIMESTAMPTZ NOT NULL
)`,
		`CREATE TABLE IF NOT EXISTS bitlocker_escrow (
  node_id TEXT NOT NULL,
  tenant_id TEXT NOT NULL,
  volume_id TEXT NOT NULL,
  key_blob TEXT NOT NULL,
  actor TEXT,
  created_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (node_id, volume_id)
)`,
	}
	for _, q := range stmts {
		if _, err := s.db.Exec(q); err != nil {
			return err
		}
	}
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM enforcement_policy`).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		p := DefaultEnforcementPolicy()
		b, _ := json.Marshal(p)
		_, err := s.db.Exec(
			`INSERT INTO enforcement_policy (id, version, mode, policy_json) VALUES (1, $1, $2, $3)`,
			p.Version, p.Mode, string(b),
		)
		return err
	}
	return nil
}

func (s *postgresStore) GetEnforcementPolicy() EnforcementPolicy {
	row := s.db.QueryRow(`SELECT policy_json FROM enforcement_policy WHERE id = 1`)
	var raw string
	if err := row.Scan(&raw); err != nil {
		return DefaultEnforcementPolicy()
	}
	var p EnforcementPolicy
	if json.Unmarshal([]byte(raw), &p) != nil {
		return DefaultEnforcementPolicy()
	}
	return p
}

func (s *postgresStore) SetEnforcementPolicy(p EnforcementPolicy, actor, detail string) error {
	if p.Version == "" {
		return fmt.Errorf("policy version required")
	}
	cur := s.GetEnforcementPolicy()
	curB, _ := json.Marshal(cur)
	newB, err := json.Marshal(p)
	if err != nil {
		return err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(
		`UPDATE enforcement_policy SET version=$1, mode=$2, policy_json=$3, prev_json=$4 WHERE id=1`,
		p.Version, p.Mode, string(newB), string(curB),
	); err != nil {
		return err
	}
	if _, err := tx.Exec(
		`INSERT INTO enforcement_history (version, mode, actor, detail, created_at) VALUES ($1,$2,$3,$4,$5)`,
		p.Version, p.Mode, actor, detail, time.Now().UTC(),
	); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *postgresStore) RollbackEnforcementPolicy(actor string) (EnforcementPolicy, bool) {
	var prev sql.NullString
	var curVer string
	row := s.db.QueryRow(`SELECT version, prev_json FROM enforcement_policy WHERE id = 1`)
	if err := row.Scan(&curVer, &prev); err != nil || !prev.Valid || prev.String == "" {
		return s.GetEnforcementPolicy(), false
	}
	var p EnforcementPolicy
	if json.Unmarshal([]byte(prev.String), &p) != nil {
		return s.GetEnforcementPolicy(), false
	}
	_ = s.SetEnforcementPolicy(p, actor, "rollback from "+curVer)
	return p, true
}

func (s *postgresStore) ListEnforcementHistory(limit int) []EnforcementPolicyRevision {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(
		`SELECT version, mode, actor, detail, created_at FROM enforcement_history ORDER BY id DESC LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []EnforcementPolicyRevision
	for rows.Next() {
		var r EnforcementPolicyRevision
		if err := rows.Scan(&r.Version, &r.Mode, &r.Actor, &r.Detail, &r.CreatedAt); err != nil {
			continue
		}
		out = append(out, r)
	}
	return out
}

func (s *postgresStore) UpsertBitlockerEscrow(e *BitlockerEscrow) {
	if e == nil {
		return
	}
	ts := time.Now().UTC()
	if !e.CreatedAt.IsZero() {
		ts = e.CreatedAt.UTC()
	}
	_, _ = s.db.Exec(
		`INSERT INTO bitlocker_escrow (node_id, tenant_id, volume_id, key_blob, actor, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6)
		 ON CONFLICT (node_id, volume_id) DO UPDATE SET key_blob=EXCLUDED.key_blob, actor=EXCLUDED.actor, created_at=EXCLUDED.created_at`,
		e.NodeID, e.TenantID, e.VolumeID, e.KeyBlob, e.Actor, ts,
	)
}

func (s *postgresStore) GetBitlockerEscrow(nodeID, volumeID string) (*BitlockerEscrow, bool) {
	row := s.db.QueryRow(
		`SELECT node_id, tenant_id, volume_id, key_blob, actor, created_at FROM bitlocker_escrow WHERE node_id=$1 AND volume_id=$2`,
		nodeID, volumeID,
	)
	var e BitlockerEscrow
	if err := row.Scan(&e.NodeID, &e.TenantID, &e.VolumeID, &e.KeyBlob, &e.Actor, &e.CreatedAt); err != nil {
		return nil, false
	}
	return &e, true
}

func (s *postgresStore) ListBitlockerEscrows(nodeID string) []BitlockerEscrowPublic {
	var rows *sql.Rows
	var err error
	if nodeID != "" {
		rows, err = s.db.Query(
			`SELECT node_id, tenant_id, volume_id, key_blob, actor, created_at FROM bitlocker_escrow WHERE node_id=$1`,
			nodeID,
		)
	} else {
		rows, err = s.db.Query(
			`SELECT node_id, tenant_id, volume_id, key_blob, actor, created_at FROM bitlocker_escrow`,
		)
	}
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []BitlockerEscrowPublic
	for rows.Next() {
		var e BitlockerEscrow
		if err := rows.Scan(&e.NodeID, &e.TenantID, &e.VolumeID, &e.KeyBlob, &e.Actor, &e.CreatedAt); err != nil {
			continue
		}
		out = append(out, e.Public())
	}
	return out
}
