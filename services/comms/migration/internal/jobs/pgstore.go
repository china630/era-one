package jobs

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// PGStore persists jobs in Postgres.
type PGStore struct {
	db *sql.DB
}

func NewPGStore(dsn string) (*PGStore, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &PGStore{db: db}, nil
}

func (p *PGStore) CreateQueued(source, mailbox string) Job {
	id := time.Now().UTC().Format("20060102150405") + "-mig-" + source
	j := Job{ID: id, Source: source, Mailbox: mailbox, Status: "queued", CreatedAt: time.Now().UTC()}
	_, _ = p.db.Exec(`INSERT INTO migration_jobs (id, source, mailbox, status, created_at, updated_at) VALUES ($1,$2,$3,'queued',now(),now())`,
		j.ID, j.Source, j.Mailbox)
	return j
}

func (p *PGStore) CreateDone(source, mailbox string, total int) Job {
	id := time.Now().UTC().Format("20060102150405") + "-mig-" + source
	j := Job{ID: id, Source: source, Mailbox: mailbox, Status: "done", ItemsTotal: total, ItemsOK: total, CreatedAt: time.Now().UTC()}
	_, _ = p.db.Exec(`INSERT INTO migration_jobs (id, source, mailbox, status, items_total, items_ok, created_at, updated_at) VALUES ($1,$2,$3,'done',$4,$4,now(),now())`,
		j.ID, j.Source, j.Mailbox, total)
	return j
}

func (p *PGStore) Get(id string) (Job, bool) {
	row := p.db.QueryRow(`SELECT id, source, mailbox, status, items_total, items_ok, items_fail, COALESCE(error,''), created_at FROM migration_jobs WHERE id=$1`, id)
	var j Job
	var errText string
	if err := row.Scan(&j.ID, &j.Source, &j.Mailbox, &j.Status, &j.ItemsTotal, &j.ItemsOK, &j.ItemsFail, &errText, &j.CreatedAt); err != nil {
		return Job{}, false
	}
	j.Error = errText
	return j, true
}

func (p *PGStore) SetStatus(id, status string) {
	_, _ = p.db.Exec(`UPDATE migration_jobs SET status=$2, updated_at=now() WHERE id=$1`, id, status)
}

func (p *PGStore) Complete(id string, total, ok, fail int) {
	_, _ = p.db.Exec(`UPDATE migration_jobs SET status='done', items_total=$2, items_ok=$3, items_fail=$4, updated_at=now() WHERE id=$1`,
		id, total, ok, fail)
}

func (p *PGStore) Fail(id, errMsg string) {
	_, _ = p.db.Exec(`UPDATE migration_jobs SET status='failed', error=$2, updated_at=now() WHERE id=$1`, id, errMsg)
}

// RequeueFailed moves failed jobs back to queued for crash resume (Phase 3).
func (p *PGStore) RequeueFailed() (int64, error) {
	res, err := p.db.Exec(`UPDATE migration_jobs SET status='queued', error=NULL, updated_at=now() WHERE status='failed'`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (p *PGStore) MarkSeen(uid string) {
	_, _ = p.db.Exec(`INSERT INTO migration_seen_uids (uid_key, seen_at) VALUES ($1, now()) ON CONFLICT DO NOTHING`, uid)
}

func (p *PGStore) Seen(uid string) bool {
	var n int
	_ = p.db.QueryRow(`SELECT 1 FROM migration_seen_uids WHERE uid_key=$1`, uid).Scan(&n)
	return n == 1
}

func (p *PGStore) Rerun(sourceUIDs []string) int {
	newItems := 0
	for _, uid := range sourceUIDs {
		if p.Seen(uid) {
			continue
		}
		p.MarkSeen(uid)
		newItems++
	}
	return newItems
}

func (p *PGStore) DequeueQueued() (Job, bool) {
	tx, err := p.db.Begin()
	if err != nil {
		return Job{}, false
	}
	defer func() { _ = tx.Rollback() }()
	row := tx.QueryRow(`SELECT id, source, mailbox, status, items_total, items_ok, items_fail, created_at FROM migration_jobs WHERE status='queued' ORDER BY created_at LIMIT 1 FOR UPDATE SKIP LOCKED`)
	var j Job
	if err := row.Scan(&j.ID, &j.Source, &j.Mailbox, &j.Status, &j.ItemsTotal, &j.ItemsOK, &j.ItemsFail, &j.CreatedAt); err != nil {
		return Job{}, false
	}
	if _, err := tx.Exec(`UPDATE migration_jobs SET status='running', updated_at=now() WHERE id=$1`, j.ID); err != nil {
		return Job{}, false
	}
	if err := tx.Commit(); err != nil {
		return Job{}, false
	}
	j.Status = "running"
	return j, true
}

// SavePayload stores job request JSON for orchestrator replay.
func (p *PGStore) SavePayload(id string, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = p.db.Exec(`UPDATE migration_jobs SET payload=$2, updated_at=now() WHERE id=$1`, id, b)
	return err
}

func (p *PGStore) Close() error { return p.db.Close() }

var _ Repository = (*PGStore)(nil)

func OpenRepositoryFromEnv(mem *Store) (Repository, error) {
	dsn := os.Getenv("ERA_MIG_POSTGRES_DSN")
	if dsn == "" {
		return mem, nil
	}
	pg, err := NewPGStore(dsn)
	if err != nil {
		return nil, fmt.Errorf("pg store: %w", err)
	}
	return pg, nil
}
