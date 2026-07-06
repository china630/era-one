// SQLite persistence для federated hub (S7-3).
package hub

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// Store — persistent SQLite backend для Hub.
type Store struct {
	db *sql.DB
}

// OpenStore открывает или создаёт SQLite БД hub.
func OpenStore(path string) (*Store, error) {
	if path == "" {
		return nil, fmt.Errorf("hub store: empty path")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS submissions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  zone_id TEXT NOT NULL,
  vector_json TEXT NOT NULL,
  sample_count INTEGER NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE TABLE IF NOT EXISTS global_model (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  round INTEGER NOT NULL,
  model_json TEXT NOT NULL
);
`)
	return err
}

// SaveSubmission сохраняет gradient submission.
func (s *Store) SaveSubmission(sub GradientSubmission) error {
	body, err := json.Marshal(sub.Vector)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(
		`INSERT INTO submissions (zone_id, vector_json, sample_count) VALUES (?, ?, ?)`,
		sub.ZoneID, string(body), sub.SampleCount,
	)
	return err
}

// SaveGlobalModel сохраняет агрегированную модель.
func (s *Store) SaveGlobalModel(round int, model []float64) error {
	body, err := json.Marshal(model)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
INSERT INTO global_model (id, round, model_json) VALUES (1, ?, ?)
ON CONFLICT(id) DO UPDATE SET round=excluded.round, model_json=excluded.model_json`,
		round, string(body),
	)
	return err
}

// LoadGlobalModel загружает последнюю глобальную модель.
func (s *Store) LoadGlobalModel() ([]float64, int, error) {
	var round int
	var raw string
	err := s.db.QueryRow(`SELECT round, model_json FROM global_model WHERE id = 1`).Scan(&round, &raw)
	if err == sql.ErrNoRows {
		return nil, 0, nil
	}
	if err != nil {
		return nil, 0, err
	}
	var model []float64
	if err := json.Unmarshal([]byte(raw), &model); err != nil {
		return nil, 0, err
	}
	return model, round, nil
}

// Close закрывает БД.
func (s *Store) Close() error {
	return s.db.Close()
}

// PersistentHub оборачивает Hub с SQLite persistence.
type PersistentHub struct {
	*Hub
	store *Store
}

// NewPersistent создаёт hub с записью в SQLite.
func NewPersistent(epsilon float64, store *Store) *PersistentHub {
	h := New(epsilon)
	if store != nil {
		if model, round, err := store.LoadGlobalModel(); err == nil && len(model) > 0 {
			h.mu.Lock()
			h.global = model
			h.round = round
			h.mu.Unlock()
		}
	}
	return &PersistentHub{Hub: h, store: store}
}

// Submit сохраняет submission в БД.
func (p *PersistentHub) Submit(sub GradientSubmission) error {
	if err := p.Hub.Submit(sub); err != nil {
		return err
	}
	if p.store != nil {
		return p.store.SaveSubmission(sub)
	}
	return nil
}

// Aggregate агрегирует и сохраняет модель.
func (p *PersistentHub) Aggregate() ([]float64, int) {
	model, round := p.Hub.Aggregate()
	if p.store != nil && len(model) > 0 {
		_ = p.store.SaveGlobalModel(round, model)
	}
	return model, round
}

// AuditEntries делегирует in-memory hub.
func (p *PersistentHub) AuditEntries() []SubmissionAudit {
	return p.Hub.AuditEntries()
}
