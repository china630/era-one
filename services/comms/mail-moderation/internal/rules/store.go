// Package rules — persist policy documents in PG.
package rules

import (
	"context"
	"database/sql"
	"os"
	"time"

	"era/services/comms/mail-moderation/internal/policy"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Store — moderation_rules YAML bodies.
type Store struct {
	db *sql.DB
}

func OpenFromEnv() (*Store, error) {
	dsn := os.Getenv("ERA_MM_POSTGRES_DSN")
	if dsn == "" {
		return nil, nil
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) SaveDocument(doc policy.Document) error {
	if s == nil {
		return nil
	}
	b, err := policy.MarshalDocument(doc)
	if err != nil {
		return err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM moderation_rules`); err != nil {
		return err
	}
	for _, r := range doc.Rules {
		one := policy.Document{Rules: []policy.Rule{r}}
		yb, err := policy.MarshalDocument(one)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT INTO moderation_rules (id, priority, yaml_body, updated_at) VALUES ($1,$2,$3,now())`,
			r.ID, r.Priority, string(yb)); err != nil {
			return err
		}
	}
	_ = b
	return tx.Commit()
}

func (s *Store) LoadDocument() (policy.Document, error) {
	if s == nil {
		return policy.Document{}, nil
	}
	rows, err := s.db.Query(`SELECT yaml_body FROM moderation_rules ORDER BY priority, id`)
	if err != nil {
		return policy.Document{}, err
	}
	defer rows.Close()
	var all []policy.Rule
	for rows.Next() {
		var yb string
		if err := rows.Scan(&yb); err != nil {
			return policy.Document{}, err
		}
		doc, err := policy.ParseDocument([]byte(yb))
		if err != nil {
			return policy.Document{}, err
		}
		all = append(all, doc.Rules...)
	}
	return policy.Document{Rules: all}, nil
}

// MemorySave — no-op helper for tests without PG.
type MemorySave struct {
	Doc policy.Document
}

func (m *MemorySave) SaveDocument(doc policy.Document) error {
	m.Doc = doc
	return nil
}

func (m *MemorySave) LoadDocument() (policy.Document, error) {
	return m.Doc, nil
}
