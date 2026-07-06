package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type postgresStore struct {
	db          *sql.DB
	tenantFilter string
}

// NewPostgres открывает Postgres backend control-plane (S6-14).
func NewPostgres(dsn string) (Repository, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("postgres ping: %w", err)
	}
	s := &postgresStore{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *postgresStore) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS assets (
  node_id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  hostname TEXT,
  platform TEXT,
  agent_id TEXT,
  agent_version TEXT,
  last_seen TIMESTAMPTZ NOT NULL
)`,
		`CREATE TABLE IF NOT EXISTS cases (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  status TEXT NOT NULL,
  tenant_id TEXT NOT NULL DEFAULT '',
  assignee TEXT,
  detection_id TEXT,
  node_id TEXT,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
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
  created_at TIMESTAMPTZ NOT NULL
)`,
		`CREATE TABLE IF NOT EXISTS case_timeline (
  id TEXT PRIMARY KEY,
  case_id TEXT NOT NULL,
  kind TEXT NOT NULL,
  actor TEXT,
  detail TEXT,
  created_at TIMESTAMPTZ NOT NULL
)`,
		`CREATE TABLE IF NOT EXISTS audit_log (
  id TEXT PRIMARY KEY,
  action TEXT NOT NULL,
  actor TEXT,
  target TEXT,
  detail TEXT,
  created_at TIMESTAMPTZ NOT NULL
)`,
	}
	for _, q := range stmts {
		if _, err := s.db.Exec(q); err != nil {
			return err
		}
	}
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM policy`).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		p := DefaultPolicy()
		b, _ := json.Marshal(p.Rules)
		_, err := s.db.Exec(`INSERT INTO policy (id, version, rules_json) VALUES (1, $1, $2)`, p.Version, string(b))
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

func (s *postgresStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *postgresStore) UpsertAsset(a *Asset) {
	s.UpsertAssetFull(a)
}

func (s *postgresStore) ListAssets() []*Asset {
	q := assetSelectSQLPG
	var args []any
	if s.tenantFilter != "" {
		q += ` WHERE tenant_id = $1`
		args = append(args, s.tenantFilter)
	}
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	return scanAssets(rows)
}

func (s *postgresStore) AssetCoverage() float64 {
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

func (s *postgresStore) CreateCase(c *Case) {
	now := nowUTC()
	c.CreatedAt = now
	c.UpdatedAt = now
	if c.Status == "" {
		c.Status = "new"
	}
	_, _ = s.db.Exec(`INSERT INTO cases(id,title,status,tenant_id,assignee,detection_id,node_id,created_at,updated_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		c.ID, c.Title, c.Status, c.TenantID, c.Assignee, c.DetectionID, c.NodeID, c.CreatedAt, c.UpdatedAt)
	ev := newTimeline(c.ID, "created", "system", "case opened")
	_, _ = s.db.Exec(`INSERT INTO case_timeline(id,case_id,kind,actor,detail,created_at) VALUES($1,$2,$3,$4,$5,$6)`,
		ev.ID, ev.CaseID, ev.Kind, ev.Actor, ev.Detail, ev.CreatedAt)
}

func (s *postgresStore) GetCase(id string) (*Case, bool) {
	row := s.db.QueryRow(`SELECT id,title,status,tenant_id,assignee,detection_id,node_id,created_at,updated_at FROM cases WHERE id=$1`, id)
	c, err := scanCase(row)
	if err != nil {
		return nil, false
	}
	return c, true
}

func (s *postgresStore) UpdateCase(id string, fn func(*Case)) (*Case, bool) {
	c, ok := s.GetCase(id)
	if !ok {
		return nil, false
	}
	fn(c)
	c.UpdatedAt = nowUTC()
	_, _ = s.db.Exec(`UPDATE cases SET title=$1, status=$2, assignee=$3, detection_id=$4, node_id=$5, updated_at=$6 WHERE id=$7`,
		c.Title, c.Status, c.Assignee, c.DetectionID, c.NodeID, c.UpdatedAt, id)
	return c, true
}

func (s *postgresStore) ListCases() []*Case {
	q := `SELECT id,title,status,tenant_id,assignee,detection_id,node_id,created_at,updated_at FROM cases`
	var args []any
	if s.tenantFilter != "" {
		q += ` WHERE tenant_id = $1`
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
		c, err := scanCase(rows)
		if err != nil {
			continue
		}
		out = append(out, c)
	}
	return out
}

func (s *postgresStore) Policy() PolicyBundle {
	var version, rulesJSON string
	err := s.db.QueryRow(`SELECT version, rules_json FROM policy WHERE id=1`).Scan(&version, &rulesJSON)
	if err != nil {
		return DefaultPolicy()
	}
	var rules map[string]string
	_ = json.Unmarshal([]byte(rulesJSON), &rules)
	return PolicyBundle{Version: version, Rules: rules}
}

func (s *postgresStore) SetPolicy(p PolicyBundle) {
	b, _ := json.Marshal(p.Rules)
	_, _ = s.db.Exec(`UPDATE policy SET version=$1, rules_json=$2 WHERE id=1`, p.Version, string(b))
}

func scanAssets(rows *sql.Rows) []*Asset {
	var out []*Asset
	for rows.Next() {
		var a Asset
		var macJ, ipJ sql.NullString
		var cpuC, ram, disk sql.NullInt64
		var invTS sql.NullTime
		var assetKind sql.NullString
		var managed sql.NullBool
		if err := rows.Scan(&a.NodeID, &a.TenantID, &a.Hostname, &a.Platform, &a.AgentID, &a.AgentVersion, &a.LastSeen,
			&a.FQDN, &a.OSName, &a.OSVersion, &a.Kernel, &a.CPUModel, &cpuC, &ram, &disk,
			&a.SerialNumber, &a.BoardSerial, &a.Manufacturer, &a.Model, &macJ, &ipJ, &invTS,
			&assetKind, &managed); err != nil {
			continue
		}
		if invTS.Valid {
			a.InventoryUpdatedAt = invTS.Time
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
			a.Managed = managed.Bool
		} else if a.AgentID != "" {
			a.Managed = true
		}
		out = append(out, &a)
	}
	return out
}

type caseScanner interface {
	Scan(dest ...any) error
}

func scanCase(row caseScanner) (*Case, error) {
	var c Case
	if err := row.Scan(&c.ID, &c.Title, &c.Status, &c.TenantID, &c.Assignee, &c.DetectionID, &c.NodeID, &c.CreatedAt, &c.UpdatedAt); err != nil {
		return nil, err
	}
	return &c, nil
}
