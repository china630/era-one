// SQLite persistence для national TAXII hub (S7-4).
package hub

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"era/services/national-hub/internal/dp"
	_ "modernc.org/sqlite"
)

// SQLiteStore — persistent TAXII object store.
type SQLiteStore struct {
	db *sql.DB
}

// OpenSQLiteStore открывает или создаёт SQLite БД TAXII.
func OpenSQLiteStore(path string) (*SQLiteStore, error) {
	if path == "" {
		return nil, fmt.Errorf("empty store path")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil && filepath.Dir(path) != "." {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &SQLiteStore{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *SQLiteStore) migrate() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS taxii_objects (
  collection TEXT NOT NULL,
  id TEXT NOT NULL,
  org_id TEXT NOT NULL,
  raw_json TEXT NOT NULL,
  created_at TEXT NOT NULL,
  PRIMARY KEY (collection, id)
);
CREATE TABLE IF NOT EXISTS taxii_subscribers (
  collection TEXT NOT NULL,
  org_id TEXT NOT NULL,
  PRIMARY KEY (collection, org_id)
);
`)
	return err
}

func (s *SQLiteStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *SQLiteStore) Publish(collection, orgID, objID string, data []byte) {
	_, _ = s.db.Exec(`INSERT OR REPLACE INTO taxii_objects(collection,id,org_id,raw_json,created_at) VALUES(?,?,?,?,?)`,
		collection, objID, orgID, string(data), time.Now().UTC().Format(time.RFC3339Nano))
}

func (s *SQLiteStore) Poll(collection string) []Object {
	rows, err := s.db.Query(`SELECT id, org_id, raw_json, created_at FROM taxii_objects WHERE collection=? ORDER BY created_at`, collection)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []Object
	for rows.Next() {
		var o Object
		var ts string
		if err := rows.Scan(&o.ID, &o.OrgID, &o.RawJSON, &ts); err != nil {
			continue
		}
		o.Data = []byte(o.RawJSON)
		o.CreatedAt, _ = time.Parse(time.RFC3339Nano, ts)
		out = append(out, o)
	}
	return out
}

func (s *SQLiteStore) Subscribe(orgID, collection string) {
	_, _ = s.db.Exec(`INSERT OR IGNORE INTO taxii_subscribers(collection, org_id) VALUES(?,?)`, collection, orgID)
}

func (s *SQLiteStore) SubscriberCount(collection string) int {
	var n int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM taxii_subscribers WHERE collection=?`, collection).Scan(&n)
	return n
}

func (s *SQLiteStore) PublishCount(collection string) int {
	var n int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM taxii_objects WHERE collection=?`, collection).Scan(&n)
	return n
}

func (s *SQLiteStore) NoisyPublishCount(collection string, epsilon float64) float64 {
	return dp.NoisyCount(s.PublishCount(collection), epsilon)
}

// ObjectStore — общий интерфейс in-memory и SQLite store.
type ObjectStore interface {
	Publish(collection, orgID, objID string, data []byte)
	Poll(collection string) []Object
	Subscribe(orgID, collection string)
	SubscriberCount(collection string) int
	PublishCount(collection string) int
	NoisyPublishCount(collection string, epsilon float64) float64
}

var (
	_ ObjectStore = (*Store)(nil)
	_ ObjectStore = (*SQLiteStore)(nil)
)
