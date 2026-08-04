package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

type sqliteStore struct {
	db *sql.DB
	mu sync.Mutex
}

func NewSQLite(path string) (Repository, error) {
	if path == "" {
		return nil, fmt.Errorf("empty store path")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && filepath.Dir(path) != "." {
		return nil, err
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
		`CREATE TABLE IF NOT EXISTS incidents (
  id TEXT PRIMARY KEY,
  tenant_id TEXT,
  title TEXT NOT NULL,
  description TEXT,
  status TEXT NOT NULL,
  priority TEXT,
  node_id TEXT,
  requester TEXT,
  assignee TEXT,
  sla_due_at TEXT,
  sla_breached INTEGER DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
)`,
		`CREATE TABLE IF NOT EXISTS service_requests (
  id TEXT PRIMARY KEY,
  tenant_id TEXT,
  title TEXT NOT NULL,
  category TEXT,
  status TEXT NOT NULL,
  node_id TEXT,
  requester TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
)`,
		`CREATE TABLE IF NOT EXISTS problems (
  id TEXT PRIMARY KEY,
  tenant_id TEXT,
  title TEXT NOT NULL,
  status TEXT NOT NULL,
  node_id TEXT,
  created_at TEXT NOT NULL
)`,
		`CREATE TABLE IF NOT EXISTS changes (
  id TEXT PRIMARY KEY,
  tenant_id TEXT,
  title TEXT NOT NULL,
  status TEXT NOT NULL,
  risk TEXT,
  node_id TEXT,
  created_at TEXT NOT NULL
)`,
		`CREATE TABLE IF NOT EXISTS comments (
  id TEXT PRIMARY KEY,
  ticket_id TEXT NOT NULL,
  kind TEXT NOT NULL,
  author TEXT NOT NULL,
  body TEXT NOT NULL,
  created_at TEXT NOT NULL
)`,
	}
	for _, q := range stmts {
		if _, err := s.db.Exec(q); err != nil {
			return err
		}
	}
	// Best-effort columns for detail/PATCH (existing DBs).
	alters := []string{
		`ALTER TABLE service_requests ADD COLUMN assignee TEXT`,
		`ALTER TABLE service_requests ADD COLUMN sla_status TEXT`,
		`ALTER TABLE problems ADD COLUMN assignee TEXT`,
		`ALTER TABLE problems ADD COLUMN sla_status TEXT`,
		`ALTER TABLE problems ADD COLUMN updated_at TEXT`,
		`ALTER TABLE changes ADD COLUMN assignee TEXT`,
		`ALTER TABLE changes ADD COLUMN sla_status TEXT`,
		`ALTER TABLE changes ADD COLUMN updated_at TEXT`,
	}
	for _, q := range alters {
		_, _ = s.db.Exec(q)
	}
	return nil
}

func (s *sqliteStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *sqliteStore) CreateIncident(i *Incident) {
	now := nowUTC()
	i.CreatedAt = now
	i.UpdatedAt = now
	if i.Status == "" {
		i.Status = StatusNew
	}
	var sla string
	if i.SLADueAt != nil {
		sla = i.SLADueAt.UTC().Format(time.RFC3339Nano)
	}
	_, _ = s.db.Exec(`INSERT INTO incidents(id,tenant_id,title,description,status,priority,node_id,requester,assignee,sla_due_at,created_at,updated_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
		i.ID, i.TenantID, i.Title, i.Description, i.Status, i.Priority, i.NodeID, i.Requester, i.Assignee, sla, i.CreatedAt.Format(time.RFC3339Nano), i.UpdatedAt.Format(time.RFC3339Nano))
}

func (s *sqliteStore) GetIncident(id string) (*Incident, bool) {
	row := s.db.QueryRow(`SELECT id,tenant_id,title,description,status,priority,node_id,requester,assignee,sla_due_at,sla_breached,created_at,updated_at FROM incidents WHERE id=?`, id)
	return scanIncident(row)
}

func scanIncident(row *sql.Row) (*Incident, bool) {
	var i Incident
	var sla, ca, ua string
	var breached int
	if err := row.Scan(&i.ID, &i.TenantID, &i.Title, &i.Description, &i.Status, &i.Priority, &i.NodeID, &i.Requester, &i.Assignee, &sla, &breached, &ca, &ua); err != nil {
		return nil, false
	}
	if sla != "" {
		t, _ := time.Parse(time.RFC3339Nano, sla)
		i.SLADueAt = &t
	}
	i.SLABreached = breached == 1
	i.CreatedAt, _ = time.Parse(time.RFC3339Nano, ca)
	i.UpdatedAt, _ = time.Parse(time.RFC3339Nano, ua)
	return &i, true
}

func (s *sqliteStore) UpdateIncident(id string, fn func(*Incident)) (*Incident, bool) {
	i, ok := s.GetIncident(id)
	if !ok {
		return nil, false
	}
	fn(i)
	i.UpdatedAt = nowUTC()
	var sla string
	if i.SLADueAt != nil {
		sla = i.SLADueAt.UTC().Format(time.RFC3339Nano)
	}
	breached := 0
	if i.SLABreached {
		breached = 1
	}
	_, _ = s.db.Exec(`UPDATE incidents SET title=?,description=?,status=?,priority=?,node_id=?,requester=?,assignee=?,sla_due_at=?,sla_breached=?,updated_at=? WHERE id=?`,
		i.Title, i.Description, i.Status, i.Priority, i.NodeID, i.Requester, i.Assignee, sla, breached, i.UpdatedAt.Format(time.RFC3339Nano), id)
	return i, true
}

func (s *sqliteStore) ListIncidents() []*Incident {
	rows, err := s.db.Query(`SELECT id,tenant_id,title,description,status,priority,node_id,requester,assignee,sla_due_at,sla_breached,created_at,updated_at FROM incidents ORDER BY updated_at DESC`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []*Incident
	for rows.Next() {
		var i Incident
		var sla, ca, ua string
		var breached int
		if rows.Scan(&i.ID, &i.TenantID, &i.Title, &i.Description, &i.Status, &i.Priority, &i.NodeID, &i.Requester, &i.Assignee, &sla, &breached, &ca, &ua) != nil {
			continue
		}
		if sla != "" {
			t, _ := time.Parse(time.RFC3339Nano, sla)
			i.SLADueAt = &t
		}
		i.SLABreached = breached == 1
		i.CreatedAt, _ = time.Parse(time.RFC3339Nano, ca)
		i.UpdatedAt, _ = time.Parse(time.RFC3339Nano, ua)
		out = append(out, &i)
	}
	return out
}

func (s *sqliteStore) CreateRequest(r *ServiceRequest) {
	now := nowUTC()
	r.CreatedAt = now
	r.UpdatedAt = now
	if r.Status == "" {
		r.Status = StatusNew
	}
	if r.SLAStatus == "" {
		r.SLAStatus = "none"
	}
	_, _ = s.db.Exec(`INSERT INTO service_requests(id,tenant_id,title,category,status,node_id,requester,created_at,updated_at,assignee,sla_status) VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
		r.ID, r.TenantID, r.Title, r.Category, r.Status, r.NodeID, r.Requester, r.CreatedAt.Format(time.RFC3339Nano), r.UpdatedAt.Format(time.RFC3339Nano), r.Assignee, r.SLAStatus)
}

func (s *sqliteStore) GetRequest(id string) (*ServiceRequest, bool) {
	row := s.db.QueryRow(`SELECT id,tenant_id,title,category,status,node_id,requester,created_at,updated_at,COALESCE(assignee,''),COALESCE(sla_status,'none') FROM service_requests WHERE id=?`, id)
	var r ServiceRequest
	var ca, ua string
	if err := row.Scan(&r.ID, &r.TenantID, &r.Title, &r.Category, &r.Status, &r.NodeID, &r.Requester, &ca, &ua, &r.Assignee, &r.SLAStatus); err != nil {
		return nil, false
	}
	r.CreatedAt, _ = time.Parse(time.RFC3339Nano, ca)
	r.UpdatedAt, _ = time.Parse(time.RFC3339Nano, ua)
	return &r, true
}

func (s *sqliteStore) UpdateRequest(id string, fn func(*ServiceRequest)) (*ServiceRequest, bool) {
	r, ok := s.GetRequest(id)
	if !ok {
		return nil, false
	}
	fn(r)
	r.UpdatedAt = nowUTC()
	_, _ = s.db.Exec(`UPDATE service_requests SET title=?,category=?,status=?,node_id=?,requester=?,assignee=?,sla_status=?,updated_at=? WHERE id=?`,
		r.Title, r.Category, r.Status, r.NodeID, r.Requester, r.Assignee, r.SLAStatus, r.UpdatedAt.Format(time.RFC3339Nano), id)
	return r, true
}

func (s *sqliteStore) ListRequests() []*ServiceRequest {
	rows, err := s.db.Query(`SELECT id,tenant_id,title,category,status,node_id,requester,created_at,updated_at,COALESCE(assignee,''),COALESCE(sla_status,'none') FROM service_requests ORDER BY updated_at DESC`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []*ServiceRequest
	for rows.Next() {
		var r ServiceRequest
		var ca, ua string
		if rows.Scan(&r.ID, &r.TenantID, &r.Title, &r.Category, &r.Status, &r.NodeID, &r.Requester, &ca, &ua, &r.Assignee, &r.SLAStatus) != nil {
			continue
		}
		r.CreatedAt, _ = time.Parse(time.RFC3339Nano, ca)
		r.UpdatedAt, _ = time.Parse(time.RFC3339Nano, ua)
		out = append(out, &r)
	}
	return out
}

func (s *sqliteStore) CreateProblem(p *Problem) {
	now := nowUTC()
	p.CreatedAt = now
	p.UpdatedAt = now
	if p.Status == "" {
		p.Status = StatusNew
	}
	if p.SLAStatus == "" {
		p.SLAStatus = "none"
	}
	_, _ = s.db.Exec(`INSERT INTO problems(id,tenant_id,title,status,node_id,created_at,assignee,sla_status,updated_at) VALUES(?,?,?,?,?,?,?,?,?)`,
		p.ID, p.TenantID, p.Title, p.Status, p.NodeID, p.CreatedAt.Format(time.RFC3339Nano), p.Assignee, p.SLAStatus, p.UpdatedAt.Format(time.RFC3339Nano))
}

func (s *sqliteStore) GetProblem(id string) (*Problem, bool) {
	row := s.db.QueryRow(`SELECT id,tenant_id,title,status,node_id,created_at,COALESCE(assignee,''),COALESCE(sla_status,'none'),COALESCE(updated_at,'') FROM problems WHERE id=?`, id)
	var p Problem
	var ca, ua string
	if err := row.Scan(&p.ID, &p.TenantID, &p.Title, &p.Status, &p.NodeID, &ca, &p.Assignee, &p.SLAStatus, &ua); err != nil {
		return nil, false
	}
	p.CreatedAt, _ = time.Parse(time.RFC3339Nano, ca)
	if ua != "" {
		p.UpdatedAt, _ = time.Parse(time.RFC3339Nano, ua)
	}
	return &p, true
}

func (s *sqliteStore) UpdateProblem(id string, fn func(*Problem)) (*Problem, bool) {
	p, ok := s.GetProblem(id)
	if !ok {
		return nil, false
	}
	fn(p)
	p.UpdatedAt = nowUTC()
	_, _ = s.db.Exec(`UPDATE problems SET title=?,status=?,node_id=?,assignee=?,sla_status=?,updated_at=? WHERE id=?`,
		p.Title, p.Status, p.NodeID, p.Assignee, p.SLAStatus, p.UpdatedAt.Format(time.RFC3339Nano), id)
	return p, true
}

func (s *sqliteStore) ListProblems() []*Problem {
	rows, err := s.db.Query(`SELECT id,tenant_id,title,status,node_id,created_at,COALESCE(assignee,''),COALESCE(sla_status,'none'),COALESCE(updated_at,'') FROM problems ORDER BY created_at DESC`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []*Problem
	for rows.Next() {
		var p Problem
		var ca, ua string
		if rows.Scan(&p.ID, &p.TenantID, &p.Title, &p.Status, &p.NodeID, &ca, &p.Assignee, &p.SLAStatus, &ua) != nil {
			continue
		}
		p.CreatedAt, _ = time.Parse(time.RFC3339Nano, ca)
		if ua != "" {
			p.UpdatedAt, _ = time.Parse(time.RFC3339Nano, ua)
		}
		out = append(out, &p)
	}
	return out
}

func (s *sqliteStore) CreateChange(c *Change) {
	now := nowUTC()
	c.CreatedAt = now
	c.UpdatedAt = now
	if c.Status == "" {
		c.Status = StatusNew
	}
	if c.SLAStatus == "" {
		c.SLAStatus = "none"
	}
	_, _ = s.db.Exec(`INSERT INTO changes(id,tenant_id,title,status,risk,node_id,created_at,assignee,sla_status,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?)`,
		c.ID, c.TenantID, c.Title, c.Status, c.Risk, c.NodeID, c.CreatedAt.Format(time.RFC3339Nano), c.Assignee, c.SLAStatus, c.UpdatedAt.Format(time.RFC3339Nano))
}

func (s *sqliteStore) GetChange(id string) (*Change, bool) {
	row := s.db.QueryRow(`SELECT id,tenant_id,title,status,risk,node_id,created_at,COALESCE(assignee,''),COALESCE(sla_status,'none'),COALESCE(updated_at,'') FROM changes WHERE id=?`, id)
	var c Change
	var ca, ua string
	if err := row.Scan(&c.ID, &c.TenantID, &c.Title, &c.Status, &c.Risk, &c.NodeID, &ca, &c.Assignee, &c.SLAStatus, &ua); err != nil {
		return nil, false
	}
	c.CreatedAt, _ = time.Parse(time.RFC3339Nano, ca)
	if ua != "" {
		c.UpdatedAt, _ = time.Parse(time.RFC3339Nano, ua)
	}
	return &c, true
}

func (s *sqliteStore) UpdateChange(id string, fn func(*Change)) (*Change, bool) {
	c, ok := s.GetChange(id)
	if !ok {
		return nil, false
	}
	fn(c)
	c.UpdatedAt = nowUTC()
	_, _ = s.db.Exec(`UPDATE changes SET title=?,status=?,risk=?,node_id=?,assignee=?,sla_status=?,updated_at=? WHERE id=?`,
		c.Title, c.Status, c.Risk, c.NodeID, c.Assignee, c.SLAStatus, c.UpdatedAt.Format(time.RFC3339Nano), id)
	return c, true
}

func (s *sqliteStore) ListChanges() []*Change {
	rows, err := s.db.Query(`SELECT id,tenant_id,title,status,risk,node_id,created_at,COALESCE(assignee,''),COALESCE(sla_status,'none'),COALESCE(updated_at,'') FROM changes ORDER BY created_at DESC`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []*Change
	for rows.Next() {
		var c Change
		var ca, ua string
		if rows.Scan(&c.ID, &c.TenantID, &c.Title, &c.Status, &c.Risk, &c.NodeID, &ca, &c.Assignee, &c.SLAStatus, &ua) != nil {
			continue
		}
		c.CreatedAt, _ = time.Parse(time.RFC3339Nano, ca)
		if ua != "" {
			c.UpdatedAt, _ = time.Parse(time.RFC3339Nano, ua)
		}
		out = append(out, &c)
	}
	return out
}

func (s *sqliteStore) AddComment(c *Comment) {
	c.CreatedAt = nowUTC()
	_, _ = s.db.Exec(`INSERT INTO comments(id,ticket_id,kind,author,body,created_at) VALUES(?,?,?,?,?,?)`,
		c.ID, c.TicketID, c.Kind, c.Author, c.Body, c.CreatedAt.Format(time.RFC3339Nano))
}

func (s *sqliteStore) ListComments(kind TicketKind, ticketID string) []*Comment {
	rows, err := s.db.Query(`SELECT id,ticket_id,kind,author,body,created_at FROM comments WHERE kind=? AND ticket_id=? ORDER BY created_at ASC`, kind, ticketID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []*Comment
	for rows.Next() {
		var c Comment
		var ca string
		if rows.Scan(&c.ID, &c.TicketID, &c.Kind, &c.Author, &c.Body, &ca) != nil {
			continue
		}
		c.CreatedAt, _ = time.Parse(time.RFC3339Nano, ca)
		out = append(out, &c)
	}
	return out
}

// DebugJSON — helper for parity tests.
func DebugJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
