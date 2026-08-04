package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"era/services/platform/httpserver"
	"era/services/platform/workspace"

	"github.com/golang-jwt/jwt/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
)

var errNotFound = errors.New("not found")

type checklistItem struct {
	ID   string `json:"id"`
	Text string `json:"text"`
	Done bool   `json:"done"`
}

type task struct {
	ID            string          `json:"id"`
	Title         string          `json:"title"`
	Board         string          `json:"board"`
	DriveObjectID string          `json:"drive_object_id,omitempty"`
	Assignee      string          `json:"assignee,omitempty"`
	DueDate       string          `json:"due_date,omitempty"`
	Labels        []string        `json:"labels,omitempty"`
	Checklist     []checklistItem `json:"checklist,omitempty"`
	// Priority: "", "p0", "p1", "p2"
	Priority string `json:"priority,omitempty"`
	// SortKey orders cards within a board column (and swimlane).
	SortKey  float64 `json:"sort_key,omitempty"`
	TenantID string  `json:"tenant_id,omitempty"`
}

func normalizePriority(p string) string {
	switch strings.ToLower(strings.TrimSpace(p)) {
	case "p0", "p1", "p2":
		return strings.ToLower(strings.TrimSpace(p))
	default:
		return ""
	}
}

func encodeLabelsJSON(labels []string) string {
	if labels == nil {
		labels = []string{}
	}
	b, err := json.Marshal(labels)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func decodeLabelsJSON(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" || s == "[]" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func encodeChecklistJSON(items []checklistItem) string {
	if items == nil {
		items = []checklistItem{}
	}
	b, err := json.Marshal(items)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func decodeChecklistJSON(s string) []checklistItem {
	s = strings.TrimSpace(s)
	if s == "" || s == "[]" {
		return nil
	}
	var out []checklistItem
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

type boardMeta struct {
	Name string `json:"name"`
}

type store interface {
	list(ctx context.Context, tenantID string) ([]task, error)
	put(ctx context.Context, t task) error
	delete(ctx context.Context, tenantID, id string) error
	getBoard(ctx context.Context, tenantID string) (boardMeta, error)
	putBoard(ctx context.Context, tenantID string, b boardMeta) error
}

type memStore struct {
	mu     sync.Mutex
	tasks  map[string]task
	boards map[string]boardMeta
}

func newMemStore() *memStore {
	return &memStore{
		tasks:  map[string]task{},
		boards: map[string]boardMeta{},
	}
}

func (m *memStore) list(_ context.Context, tenantID string) ([]task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]task, 0, len(m.tasks))
	for _, t := range m.tasks {
		if tenantID == "" || t.TenantID == tenantID {
			out = append(out, t)
		}
	}
	return out, nil
}

func (m *memStore) put(_ context.Context, t task) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tasks[t.ID] = t
	return nil
}

func (m *memStore) delete(_ context.Context, tenantID, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tasks[id]
	if !ok || (tenantID != "" && t.TenantID != tenantID) {
		return errNotFound
	}
	delete(m.tasks, id)
	return nil
}

func (m *memStore) getBoard(_ context.Context, tenantID string) (boardMeta, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if b, ok := m.boards[tenantID]; ok {
		return b, nil
	}
	return boardMeta{Name: "Board"}, nil
}

func (m *memStore) putBoard(_ context.Context, tenantID string, b boardMeta) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if strings.TrimSpace(b.Name) == "" {
		b.Name = "Board"
	}
	m.boards[tenantID] = b
	return nil
}

type pgStore struct {
	db *sql.DB
}

func (p *pgStore) list(ctx context.Context, tenantID string) ([]task, error) {
	rows, err := p.db.QueryContext(ctx,
		`SELECT id, title, board, COALESCE(drive_object_id,''),
		        COALESCE(assignee,''), COALESCE(due_date,''),
		        COALESCE(labels_json,'[]'), COALESCE(checklist_json,'[]'),
		        COALESCE(priority,''), COALESCE(sort_key,0), tenant_id
		 FROM era_platform.projects_tasks WHERE tenant_id=$1
		 ORDER BY sort_key, id`, tenantID)
	if err != nil {
		// Pre-priority fallback (W2 schema).
		rowsW2, errW2 := p.db.QueryContext(ctx,
			`SELECT id, title, board, COALESCE(drive_object_id,''),
			        COALESCE(assignee,''), COALESCE(due_date,''),
			        COALESCE(labels_json,'[]'), COALESCE(checklist_json,'[]'), tenant_id
			 FROM era_platform.projects_tasks WHERE tenant_id=$1 ORDER BY id`, tenantID)
		if errW2 != nil {
			rows2, err2 := p.db.QueryContext(ctx,
				`SELECT id, title, board, COALESCE(drive_object_id,''),
				        COALESCE(assignee,''), COALESCE(due_date,''), tenant_id
				 FROM era_platform.projects_tasks WHERE tenant_id=$1 ORDER BY id`, tenantID)
			if err2 != nil {
				rows3, err3 := p.db.QueryContext(ctx,
					`SELECT id, title, board, COALESCE(drive_object_id,''), tenant_id
					 FROM era_platform.projects_tasks WHERE tenant_id=$1 ORDER BY id`, tenantID)
				if err3 != nil {
					return nil, err
				}
				defer rows3.Close()
				var out []task
				for rows3.Next() {
					var t task
					if err := rows3.Scan(&t.ID, &t.Title, &t.Board, &t.DriveObjectID, &t.TenantID); err != nil {
						return nil, err
					}
					out = append(out, t)
				}
				return out, rows3.Err()
			}
			defer rows2.Close()
			var out []task
			for rows2.Next() {
				var t task
				if err := rows2.Scan(&t.ID, &t.Title, &t.Board, &t.DriveObjectID, &t.Assignee, &t.DueDate, &t.TenantID); err != nil {
					return nil, err
				}
				out = append(out, t)
			}
			return out, rows2.Err()
		}
		defer rowsW2.Close()
		var out []task
		for rowsW2.Next() {
			var t task
			var labelsJSON, checklistJSON string
			if err := rowsW2.Scan(&t.ID, &t.Title, &t.Board, &t.DriveObjectID, &t.Assignee, &t.DueDate,
				&labelsJSON, &checklistJSON, &t.TenantID); err != nil {
				return nil, err
			}
			t.Labels = decodeLabelsJSON(labelsJSON)
			t.Checklist = decodeChecklistJSON(checklistJSON)
			out = append(out, t)
		}
		return out, rowsW2.Err()
	}
	defer rows.Close()
	var out []task
	for rows.Next() {
		var t task
		var labelsJSON, checklistJSON string
		if err := rows.Scan(&t.ID, &t.Title, &t.Board, &t.DriveObjectID, &t.Assignee, &t.DueDate,
			&labelsJSON, &checklistJSON, &t.Priority, &t.SortKey, &t.TenantID); err != nil {
			return nil, err
		}
		t.Priority = normalizePriority(t.Priority)
		t.Labels = decodeLabelsJSON(labelsJSON)
		t.Checklist = decodeChecklistJSON(checklistJSON)
		out = append(out, t)
	}
	return out, rows.Err()
}

func (p *pgStore) put(ctx context.Context, t task) error {
	t.Priority = normalizePriority(t.Priority)
	labelsJSON := encodeLabelsJSON(t.Labels)
	checklistJSON := encodeChecklistJSON(t.Checklist)
	_, err := p.db.ExecContext(ctx,
		`INSERT INTO era_platform.projects_tasks (id, title, board, drive_object_id, assignee, due_date, labels_json, checklist_json, priority, sort_key, tenant_id)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		 ON CONFLICT (id) DO UPDATE SET title=EXCLUDED.title, board=EXCLUDED.board,
		   drive_object_id=EXCLUDED.drive_object_id, assignee=EXCLUDED.assignee, due_date=EXCLUDED.due_date,
		   labels_json=EXCLUDED.labels_json, checklist_json=EXCLUDED.checklist_json,
		   priority=EXCLUDED.priority, sort_key=EXCLUDED.sort_key`,
		t.ID, t.Title, t.Board, t.DriveObjectID, t.Assignee, t.DueDate, labelsJSON, checklistJSON, t.Priority, t.SortKey, t.TenantID)
	if err != nil {
		_, errW2 := p.db.ExecContext(ctx,
			`INSERT INTO era_platform.projects_tasks (id, title, board, drive_object_id, assignee, due_date, labels_json, checklist_json, tenant_id)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
			 ON CONFLICT (id) DO UPDATE SET title=EXCLUDED.title, board=EXCLUDED.board,
			   drive_object_id=EXCLUDED.drive_object_id, assignee=EXCLUDED.assignee, due_date=EXCLUDED.due_date,
			   labels_json=EXCLUDED.labels_json, checklist_json=EXCLUDED.checklist_json`,
			t.ID, t.Title, t.Board, t.DriveObjectID, t.Assignee, t.DueDate, labelsJSON, checklistJSON, t.TenantID)
		if errW2 != nil {
			_, err2 := p.db.ExecContext(ctx,
				`INSERT INTO era_platform.projects_tasks (id, title, board, drive_object_id, assignee, due_date, tenant_id)
				 VALUES ($1,$2,$3,$4,$5,$6,$7)
				 ON CONFLICT (id) DO UPDATE SET title=EXCLUDED.title, board=EXCLUDED.board,
				   drive_object_id=EXCLUDED.drive_object_id, assignee=EXCLUDED.assignee, due_date=EXCLUDED.due_date`,
				t.ID, t.Title, t.Board, t.DriveObjectID, t.Assignee, t.DueDate, t.TenantID)
			if err2 != nil {
				_, err3 := p.db.ExecContext(ctx,
					`INSERT INTO era_platform.projects_tasks (id, title, board, drive_object_id, tenant_id)
					 VALUES ($1,$2,$3,$4,$5)
					 ON CONFLICT (id) DO UPDATE SET title=EXCLUDED.title, board=EXCLUDED.board,
					   drive_object_id=EXCLUDED.drive_object_id`,
					t.ID, t.Title, t.Board, t.DriveObjectID, t.TenantID)
				return err3
			}
			return nil
		}
		return nil
	}
	return nil
}

func (p *pgStore) delete(ctx context.Context, tenantID, id string) error {
	res, err := p.db.ExecContext(ctx,
		`DELETE FROM era_platform.projects_tasks WHERE id=$1 AND tenant_id=$2`, id, tenantID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errNotFound
	}
	return nil
}

func (p *pgStore) getBoard(ctx context.Context, tenantID string) (boardMeta, error) {
	var name string
	err := p.db.QueryRowContext(ctx,
		`SELECT name FROM era_platform.projects_boards WHERE tenant_id=$1`, tenantID).Scan(&name)
	if err == sql.ErrNoRows {
		return boardMeta{Name: "Board"}, nil
	}
	if err != nil {
		return boardMeta{Name: "Board"}, nil
	}
	return boardMeta{Name: name}, nil
}

func (p *pgStore) putBoard(ctx context.Context, tenantID string, b boardMeta) error {
	if strings.TrimSpace(b.Name) == "" {
		b.Name = "Board"
	}
	_, err := p.db.ExecContext(ctx,
		`INSERT INTO era_platform.projects_boards (tenant_id, name, updated_at)
		 VALUES ($1,$2,now())
		 ON CONFLICT (tenant_id) DO UPDATE SET name=EXCLUDED.name, updated_at=now()`,
		tenantID, b.Name)
	return err
}

type server struct {
	store     store
	jwtSecret []byte
	licenseOK bool
	drive     erajDrive
	eraj      *erajSessionCache
}

func licenseFromEnv() bool {
	if envTruthy("ERA_LICENSE_STRICT") || envTruthy("ERA_PRODUCTION") {
		return envTruthy("ERA_LICENSE_OFFICE_PROJECTS")
	}
	if envTruthy("ERA_OFFICE_DEV") {
		return true
	}
	return envTruthy("ERA_LICENSE_OFFICE_PROJECTS")
}

func envTruthy(k string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(k)))
	return v == "1" || v == "true" || v == "yes"
}

func newMux(s *server) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","service":"docs-projects"}`))
	})
	// Legacy tenant-scoped board (pre-.eraj). Exact paths beat /api/v1/projects/.
	mux.HandleFunc("/api/v1/projects/board", s.withAuth(s.handleBoard))
	mux.HandleFunc("/api/v1/projects/tasks", s.withAuth(s.handleTasks))
	mux.HandleFunc("/api/v1/projects/tasks/", s.withAuth(s.handleTaskByID))
	// Drive-native .eraj project boards.
	mux.HandleFunc("/api/v1/projects", s.withAuth(s.handleProjectsRoot))
	mux.HandleFunc("/api/v1/projects/", s.withAuth(s.handleProjectSub))
	return mux
}

func (s *server) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.licenseOK {
			http.Error(w, "office-projects license required", http.StatusForbidden)
			return
		}
		auth := r.Header.Get("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		tokStr := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		tok, err := jwt.Parse(tokStr, func(t *jwt.Token) (any, error) {
			if t.Method != jwt.SigningMethodHS256 {
				return nil, fmt.Errorf("alg")
			}
			return s.jwtSecret, nil
		}, jwt.WithValidMethods([]string{"HS256"}))
		if err != nil || !tok.Valid {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		claims, _ := tok.Claims.(jwt.MapClaims)
		tenant, _ := claims["tenant_id"].(string)
		sub, _ := claims["sub"].(string)
		if tenant == "" || sub == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), ctxTenant, tenant)
		ctx = context.WithValue(ctx, ctxUser, sub)
		next(w, r.WithContext(ctx))
	}
}

type ctxKey string

const ctxTenant ctxKey = "tenant"
const ctxUser ctxKey = "user"

func (s *server) handleBoard(w http.ResponseWriter, r *http.Request) {
	tenant, _ := r.Context().Value(ctxTenant).(string)
	switch r.Method {
	case http.MethodGet:
		b, err := s.store.getBoard(r.Context(), tenant)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(b)
	case http.MethodPut, http.MethodPost:
		var b boardMeta
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		if err := s.store.putBoard(r.Context(), tenant, b); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(b)
	default:
		http.Error(w, "method", 405)
	}
}

func (s *server) handleTasks(w http.ResponseWriter, r *http.Request) {
	tenant, _ := r.Context().Value(ctxTenant).(string)
	switch r.Method {
	case http.MethodGet:
		list, err := s.store.list(r.Context(), tenant)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(list)
	case http.MethodPost:
		var t task
		if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		if t.ID == "" {
			t.ID = fmt.Sprintf("task-%d", time.Now().UnixNano())
		}
		if t.Board == "" {
			t.Board = "backlog"
		}
		t.Priority = normalizePriority(t.Priority)
		t.TenantID = tenant
		if err := s.store.put(r.Context(), t); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(t)
	default:
		http.Error(w, "method", 405)
	}
}

func (s *server) handleTaskByID(w http.ResponseWriter, r *http.Request) {
	tenant, _ := r.Context().Value(ctxTenant).(string)
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/projects/tasks/")
	id = strings.Trim(id, "/")
	if id == "" || strings.Contains(id, "/") {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodDelete:
		if err := s.store.delete(r.Context(), tenant, id); err != nil {
			if errors.Is(err, errNotFound) {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), 500)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method", 405)
	}
}

func openStore() store {
	dsn := strings.TrimSpace(os.Getenv("ERA_OFFICE_DATABASE_URL"))
	if dsn == "" {
		log.Printf("docs-projects: memory store (no ERA_OFFICE_DATABASE_URL)")
		return newMemStore()
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Printf("docs-projects: postgres open failed, memory fallback: %v", err)
		return newMemStore()
	}
	if err := db.Ping(); err != nil {
		log.Printf("docs-projects: postgres ping failed, memory fallback: %v", err)
		return newMemStore()
	}
	log.Printf("docs-projects: postgres store")
	return &pgStore{db: db}
}

func (s *server) handleProjectsRoot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	if s.drive == nil {
		http.Error(w, "drive bind unavailable (ERA_DRIVE_API_URL)", http.StatusServiceUnavailable)
		return
	}
	tenant, _ := r.Context().Value(ctxTenant).(string)
	user, _ := r.Context().Value(ctxUser).(string)
	var req struct {
		Name string `json:"name"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	doc := emptyEraj(req.Name, tenant)
	id, err := s.drive.putEraj(tenant, user, doc.Name, doc, "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	doc.DriveObjectID = id
	s.cachePut(doc)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"drive_object_id": id,
		"name":            doc.Name,
		"format":          "eraj",
		"tasks":           doc.Tasks,
	})
}

func (s *server) handleProjectSub(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/projects/")
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	// Reserved legacy segments must not be treated as object ids.
	if parts[0] == "board" || parts[0] == "tasks" {
		http.NotFound(w, r)
		return
	}
	id := parts[0]
	switch {
	case len(parts) == 1:
		s.handleErajProject(w, r, id)
	case len(parts) == 2 && parts[1] == "tasks":
		s.handleErajTasks(w, r, id)
	case len(parts) == 3 && parts[1] == "tasks":
		s.handleErajTaskByID(w, r, id, parts[2])
	default:
		http.NotFound(w, r)
	}
}

func (s *server) handleErajProject(w http.ResponseWriter, r *http.Request, id string) {
	tenant, _ := r.Context().Value(ctxTenant).(string)
	user, _ := r.Context().Value(ctxUser).(string)
	switch r.Method {
	case http.MethodGet:
		doc, err := s.loadEraj(r.Context(), tenant, user, id)
		if err != nil {
			writeErajErr(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(doc)
	case http.MethodPut, http.MethodPost:
		doc, err := s.loadEraj(r.Context(), tenant, user, id)
		if err != nil {
			writeErajErr(w, err)
			return
		}
		var body struct {
			Name  string `json:"name"`
			Tasks []task `json:"tasks"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(body.Name) != "" {
			doc.Name = body.Name
			if !strings.HasSuffix(strings.ToLower(doc.Name), ".eraj") {
				doc.Name += ".eraj"
			}
		}
		if body.Tasks != nil {
			doc.Tasks = body.Tasks
			for i := range doc.Tasks {
				doc.Tasks[i].TenantID = tenant
			}
		}
		if err := s.flushEraj(tenant, user, doc); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(doc)
	default:
		http.Error(w, "method", http.StatusMethodNotAllowed)
	}
}

func (s *server) handleErajTasks(w http.ResponseWriter, r *http.Request, projectID string) {
	tenant, _ := r.Context().Value(ctxTenant).(string)
	user, _ := r.Context().Value(ctxUser).(string)
	doc, err := s.loadEraj(r.Context(), tenant, user, projectID)
	if err != nil {
		writeErajErr(w, err)
		return
	}
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(doc.Tasks)
	case http.MethodPost:
		var t task
		if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if t.ID == "" {
			t.ID = fmt.Sprintf("task-%d", time.Now().UnixNano())
		}
		if t.Board == "" {
			t.Board = "backlog"
		}
		t.Priority = normalizePriority(t.Priority)
		t.TenantID = tenant
		replaced := false
		for i := range doc.Tasks {
			if doc.Tasks[i].ID == t.ID {
				doc.Tasks[i] = t
				replaced = true
				break
			}
		}
		if !replaced {
			doc.Tasks = append(doc.Tasks, t)
		}
		if err := s.flushEraj(tenant, user, doc); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(t)
	default:
		http.Error(w, "method", http.StatusMethodNotAllowed)
	}
}

func (s *server) handleErajTaskByID(w http.ResponseWriter, r *http.Request, projectID, taskID string) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	tenant, _ := r.Context().Value(ctxTenant).(string)
	user, _ := r.Context().Value(ctxUser).(string)
	doc, err := s.loadEraj(r.Context(), tenant, user, projectID)
	if err != nil {
		writeErajErr(w, err)
		return
	}
	found := false
	next := make([]task, 0, len(doc.Tasks))
	for _, t := range doc.Tasks {
		if t.ID == taskID {
			found = true
			continue
		}
		next = append(next, t)
	}
	if !found {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	doc.Tasks = next
	if err := s.flushEraj(tenant, user, doc); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) cachePut(doc ErajProject) {
	if s.eraj == nil {
		s.eraj = newErajSessionCache()
	}
	cp := doc
	if cp.Tasks == nil {
		cp.Tasks = []task{}
	}
	s.eraj.mu.Lock()
	s.eraj.docs[doc.DriveObjectID] = &cp
	s.eraj.mu.Unlock()
}

func (s *server) loadEraj(_ context.Context, tenant, user, id string) (ErajProject, error) {
	if s.eraj != nil {
		s.eraj.mu.Lock()
		if d, ok := s.eraj.docs[id]; ok {
			cp := *d
			tasks := make([]task, len(d.Tasks))
			copy(tasks, d.Tasks)
			cp.Tasks = tasks
			s.eraj.mu.Unlock()
			return cp, nil
		}
		s.eraj.mu.Unlock()
	}
	if s.drive == nil {
		return ErajProject{}, fmt.Errorf("drive bind unavailable")
	}
	doc, err := s.drive.getEraj(tenant, user, id)
	if err != nil {
		return ErajProject{}, err
	}
	doc.DriveObjectID = id
	if doc.TenantID == "" {
		doc.TenantID = tenant
	}
	s.cachePut(doc)
	return doc, nil
}

func (s *server) flushEraj(tenant, user string, doc ErajProject) error {
	if s.drive == nil {
		return fmt.Errorf("drive bind unavailable")
	}
	doc.Format = "eraj"
	doc.TenantID = tenant
	id, err := s.drive.putEraj(tenant, user, doc.Name, doc, doc.DriveObjectID)
	if err != nil {
		return err
	}
	doc.DriveObjectID = id
	s.cachePut(doc)
	return nil
}

func writeErajErr(w http.ResponseWriter, err error) {
	if errors.Is(err, errNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	msg := err.Error()
	if strings.Contains(msg, "drive get 404") || strings.HasSuffix(msg, "404") {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	http.Error(w, msg, http.StatusBadGateway)
}

func main() {
	drv, err := newDriveClientFromEnv()
	if err != nil {
		log.Printf("docs-projects: .eraj Drive bind disabled: %v", err)
	}
	s := &server{
		store:     openStore(),
		jwtSecret: []byte(workspace.Env("ERA_IDENTITY_JWT_SECRET", "dev-only-change-in-prod")),
		licenseOK: licenseFromEnv(),
		drive:     drv,
		eraj:      newErajSessionCache(),
	}
	addr := workspace.Env("ERA_PROJECTS_HTTP_ADDR", ":8145")
	log.Printf("docs-projects listening %s (license=%v drive=%v)", addr, s.licenseOK, drv != nil)
	log.Fatal(httpserver.Listen(addr, newMux(s)))
}
