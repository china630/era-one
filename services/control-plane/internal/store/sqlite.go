package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type sqliteStore struct {
	db           *sql.DB
	tenantFilter  string
}

func NewSQLite(path string) (Repository, error) {
	if err := validateStorePath(path); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && !os.IsExist(err) {
		// allow relative path in cwd without dir
		if filepath.Dir(path) != "." {
			return nil, fmt.Errorf("mkdir store: %w", err)
		}
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &sqliteStore{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *sqliteStore) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS assets (
  node_id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  hostname TEXT,
  platform TEXT,
  agent_id TEXT,
  agent_version TEXT,
  last_seen TEXT NOT NULL
)`,
		`CREATE TABLE IF NOT EXISTS cases (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  status TEXT NOT NULL,
  tenant_id TEXT NOT NULL DEFAULT '',
  assignee TEXT,
  detection_id TEXT,
  node_id TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
)`,
		`CREATE TABLE IF NOT EXISTS policy (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  version TEXT NOT NULL,
  rules_json TEXT NOT NULL
)`,
		`CREATE TABLE IF NOT EXISTS case_notes (
  id TEXT PRIMARY KEY,
  case_id TEXT NOT NULL,
  author TEXT NOT NULL,
  body TEXT NOT NULL,
  created_at TEXT NOT NULL
)`,
		`CREATE TABLE IF NOT EXISTS case_timeline (
  id TEXT PRIMARY KEY,
  case_id TEXT NOT NULL,
  kind TEXT NOT NULL,
  actor TEXT,
  detail TEXT,
  created_at TEXT NOT NULL
)`,
		`CREATE TABLE IF NOT EXISTS audit_log (
  id TEXT PRIMARY KEY,
  action TEXT NOT NULL,
  actor TEXT,
  target TEXT,
  detail TEXT,
  created_at TEXT NOT NULL
)`,
	}
	for _, q := range stmts {
		if _, err := s.db.Exec(q); err != nil {
			return err
		}
	}
	_, _ = s.db.Exec(`ALTER TABLE cases ADD COLUMN tenant_id TEXT NOT NULL DEFAULT ''`)
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM policy`).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		p := DefaultPolicy()
		b, _ := json.Marshal(p.Rules)
		_, err := s.db.Exec(`INSERT INTO policy (id, version, rules_json) VALUES (1, ?, ?)`, p.Version, string(b))
		if err != nil {
			return err
		}
	}
	if err := s.hybridMigrate(); err != nil {
		return err
	}
	if err := s.cmdbMigrate(); err != nil {
		return err
	}
	if err := s.enforcementMigrate(); err != nil {
		return err
	}
	return s.deployMigrate()
}

func (s *sqliteStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *sqliteStore) UpsertAsset(a *Asset) {
	s.UpsertAssetFull(a)
}

func (s *sqliteStore) ListAssets() []*Asset {
	q := assetSelectSQL
	var args []any
	if s.tenantFilter != "" {
		q += ` WHERE tenant_id = ?`
		args = append(args, s.tenantFilter)
	}
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []*Asset
	for rows.Next() {
		var a Asset
		var ts, invTS, macJ, ipJ sql.NullString
		var cpuC, ram, disk, managed sql.NullInt64
		var assetKind sql.NullString
		if err := rows.Scan(&a.NodeID, &a.TenantID, &a.Hostname, &a.Platform, &a.AgentID, &a.AgentVersion, &ts,
			&a.FQDN, &a.OSName, &a.OSVersion, &a.Kernel, &a.CPUModel, &cpuC, &ram, &disk,
			&a.SerialNumber, &a.BoardSerial, &a.Manufacturer, &a.Model, &macJ, &ipJ, &invTS,
			&assetKind, &managed); err != nil {
			continue
		}
		a.LastSeen, _ = time.Parse(time.RFC3339Nano, ts.String)
		if invTS.Valid {
			a.InventoryUpdatedAt, _ = time.Parse(time.RFC3339Nano, invTS.String)
		}
		if cpuC.Valid {
			a.CPUCores = uint32(cpuC.Int64)
		}
		if ram.Valid {
			a.RAMMB = uint64(ram.Int64)
		}
		if disk.Valid {
			a.DiskTotalGB = uint64(disk.Int64)
		}
		if macJ.Valid {
			_ = json.Unmarshal([]byte(macJ.String), &a.MACAddrs)
		}
		if ipJ.Valid {
			_ = json.Unmarshal([]byte(ipJ.String), &a.IPAddrs)
		}
		if assetKind.Valid {
			a.AssetKind = assetKind.String
		}
		if managed.Valid {
			a.Managed = managed.Int64 != 0
		} else if a.AgentID != "" {
			a.Managed = true
		}
		out = append(out, &a)
	}
	return out
}

func (s *sqliteStore) AssetCoverage() float64 {
	list := s.ListAssets()
	if len(list) == 0 {
		return 0
	}
	active := 0
	cutoff := nowUTC().Add(-15 * time.Minute)
	for _, a := range list {
		if a.LastSeen.After(cutoff) {
			active++
		}
	}
	return float64(active) / float64(len(list))
}

func (s *sqliteStore) CreateCase(c *Case) {
	now := nowUTC()
	c.CreatedAt = now
	c.UpdatedAt = now
	if c.Status == "" {
		c.Status = "new"
	}
	_, _ = s.db.Exec(`INSERT INTO cases(id,title,status,tenant_id,assignee,detection_id,node_id,created_at,updated_at)
		VALUES(?,?,?,?,?,?,?,?,?)`,
		c.ID, c.Title, c.Status, c.TenantID, c.Assignee, c.DetectionID, c.NodeID,
		c.CreatedAt.Format(time.RFC3339Nano), c.UpdatedAt.Format(time.RFC3339Nano))
	ev := newTimeline(c.ID, "created", "system", "case opened")
	_, _ = s.db.Exec(`INSERT INTO case_timeline(id,case_id,kind,actor,detail,created_at) VALUES(?,?,?,?,?,?)`,
		ev.ID, ev.CaseID, ev.Kind, ev.Actor, ev.Detail, ev.CreatedAt.Format(time.RFC3339Nano))
}

func (s *sqliteStore) GetCase(id string) (*Case, bool) {
	row := s.db.QueryRow(`SELECT id,title,status,tenant_id,assignee,detection_id,node_id,created_at,updated_at FROM cases WHERE id=?`, id)
	var c Case
	var ca, ua string
	if err := row.Scan(&c.ID, &c.Title, &c.Status, &c.TenantID, &c.Assignee, &c.DetectionID, &c.NodeID, &ca, &ua); err != nil {
		return nil, false
	}
	c.CreatedAt, _ = time.Parse(time.RFC3339Nano, ca)
	c.UpdatedAt, _ = time.Parse(time.RFC3339Nano, ua)
	return &c, true
}

func (s *sqliteStore) UpdateCase(id string, fn func(*Case)) (*Case, bool) {
	c, ok := s.GetCase(id)
	if !ok {
		return nil, false
	}
	fn(c)
	c.UpdatedAt = nowUTC()
	_, _ = s.db.Exec(`UPDATE cases SET title=?, status=?, assignee=?, detection_id=?, node_id=?, updated_at=? WHERE id=?`,
		c.Title, c.Status, c.Assignee, c.DetectionID, c.NodeID, c.UpdatedAt.Format(time.RFC3339Nano), id)
	return c, true
}

func (s *sqliteStore) ListCases() []*Case {
	q := `SELECT id,title,status,tenant_id,assignee,detection_id,node_id,created_at,updated_at FROM cases`
	var args []any
	if s.tenantFilter != "" {
		q += ` WHERE tenant_id = ?`
		args = append(args, s.tenantFilter)
	}
	q += ` ORDER BY updated_at DESC`
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []*Case
	for rows.Next() {
		var c Case
		var ca, ua string
		if err := rows.Scan(&c.ID, &c.Title, &c.Status, &c.TenantID, &c.Assignee, &c.DetectionID, &c.NodeID, &ca, &ua); err != nil {
			continue
		}
		c.CreatedAt, _ = time.Parse(time.RFC3339Nano, ca)
		c.UpdatedAt, _ = time.Parse(time.RFC3339Nano, ua)
		out = append(out, &c)
	}
	return out
}

func (s *sqliteStore) Policy() PolicyBundle {
	var version, rulesJSON string
	err := s.db.QueryRow(`SELECT version, rules_json FROM policy WHERE id=1`).Scan(&version, &rulesJSON)
	if err != nil {
		return DefaultPolicy()
	}
	var rules map[string]string
	_ = json.Unmarshal([]byte(rulesJSON), &rules)
	return PolicyBundle{Version: version, Rules: rules}
}

func (s *sqliteStore) SetPolicy(p PolicyBundle) {
	b, _ := json.Marshal(p.Rules)
	_, _ = s.db.Exec(`UPDATE policy SET version=?, rules_json=? WHERE id=1`, p.Version, string(b))
}
