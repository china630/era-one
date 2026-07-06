package store

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

func (s *sqliteStore) hybridMigrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS hybrid_policy (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  json TEXT NOT NULL
)`,
		`CREATE TABLE IF NOT EXISTS hybrid_runtime (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  json TEXT NOT NULL
)`,
		`CREATE TABLE IF NOT EXISTS hybrid_lease (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  token TEXT NOT NULL DEFAULT '',
  renewed_at TEXT
)`,
		`CREATE TABLE IF NOT EXISTS egress_audit (
  id TEXT PRIMARY KEY,
  at TEXT NOT NULL,
  kind TEXT NOT NULL,
  target TEXT,
  level TEXT,
  bytes INTEGER,
  payload_hash TEXT
)`,
	}
	for _, q := range stmts {
		if _, err := s.db.Exec(q); err != nil {
			return err
		}
	}
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM hybrid_policy`).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		b, _ := json.Marshal(DefaultHybridPolicy())
		_, err := s.db.Exec(`INSERT INTO hybrid_policy (id, json) VALUES (1, ?)`, string(b))
		if err != nil {
			return err
		}
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM hybrid_runtime`).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		b, _ := json.Marshal(HybridRuntime{})
		_, err := s.db.Exec(`INSERT INTO hybrid_runtime (id, json) VALUES (1, ?)`, string(b))
		return err
	}
	return nil
}

func (s *sqliteStore) GetHybridPolicy() HybridPolicy {
	var raw string
	if err := s.db.QueryRow(`SELECT json FROM hybrid_policy WHERE id=1`).Scan(&raw); err != nil {
		return DefaultHybridPolicy()
	}
	var p HybridPolicy
	if json.Unmarshal([]byte(raw), &p) != nil {
		return DefaultHybridPolicy()
	}
	return p
}

func (s *sqliteStore) SetHybridPolicy(p HybridPolicy) {
	b, _ := json.Marshal(p)
	_, _ = s.db.Exec(`UPDATE hybrid_policy SET json=? WHERE id=1`, string(b))
}

func (s *sqliteStore) GetHybridRuntime() HybridRuntime {
	var raw string
	if err := s.db.QueryRow(`SELECT json FROM hybrid_runtime WHERE id=1`).Scan(&raw); err != nil {
		return HybridRuntime{}
	}
	var r HybridRuntime
	_ = json.Unmarshal([]byte(raw), &r)
	return r
}

func (s *sqliteStore) SetHybridRuntime(r HybridRuntime) {
	b, _ := json.Marshal(r)
	_, _ = s.db.Exec(`UPDATE hybrid_runtime SET json=? WHERE id=1`, string(b))
}

func (s *sqliteStore) RecordEgressAudit(e *EgressAuditEntry) {
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	if e.At.IsZero() {
		e.At = nowUTC()
	}
	_, _ = s.db.Exec(`INSERT INTO egress_audit (id, at, kind, target, level, bytes, payload_hash) VALUES (?,?,?,?,?,?,?)`,
		e.ID, e.At.Format(time.RFC3339), e.Kind, e.Target, e.Level, e.Bytes, e.PayloadHash)
}

func (s *sqliteStore) ListEgressAudit(limit int) []*EgressAuditEntry {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.Query(`SELECT id, at, kind, target, level, bytes, payload_hash FROM egress_audit ORDER BY at DESC LIMIT ?`, limit)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []*EgressAuditEntry
	for rows.Next() {
		var e EgressAuditEntry
		var at string
		if rows.Scan(&e.ID, &at, &e.Kind, &e.Target, &e.Level, &e.Bytes, &e.PayloadHash) == nil {
			e.At, _ = time.Parse(time.RFC3339, at)
			out = append(out, &e)
		}
	}
	return out
}

func (s *sqliteStore) GetLeaseCache() (string, time.Time) {
	var token, at string
	if err := s.db.QueryRow(`SELECT token, renewed_at FROM hybrid_lease WHERE id=1`).Scan(&token, &at); err != nil {
		return "", time.Time{}
	}
	t, _ := time.Parse(time.RFC3339, at)
	return token, t
}

func (s *sqliteStore) SetLeaseCache(token string, lastRenew time.Time) {
	if lastRenew.IsZero() {
		lastRenew = nowUTC()
	}
	var n int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM hybrid_lease`).Scan(&n)
	if n == 0 {
		_, _ = s.db.Exec(`INSERT INTO hybrid_lease (id, token, renewed_at) VALUES (1, ?, ?)`, token, lastRenew.Format(time.RFC3339))
		return
	}
	_, _ = s.db.Exec(`UPDATE hybrid_lease SET token=?, renewed_at=? WHERE id=1`, token, lastRenew.Format(time.RFC3339))
}
