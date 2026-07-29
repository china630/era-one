package hold

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// PGStore — hold в Postgres (moderation_holds).
type PGStore struct {
	db  *sql.DB
	now func() time.Time
}

// OpenPGFromEnv открывает PG при ERA_MM_POSTGRES_DSN; иначе nil.
func OpenPGFromEnv() (*PGStore, error) {
	dsn := os.Getenv("ERA_MM_POSTGRES_DSN")
	if dsn == "" {
		return nil, nil
	}
	return OpenPG(dsn)
}

func OpenPG(dsn string) (*PGStore, error) {
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
	return &PGStore{db: db, now: time.Now}, nil
}

func (s *PGStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *PGStore) Put(r Record) (Record, error) {
	if r.ID == "" {
		id, err := newID()
		if err != nil {
			return Record{}, err
		}
		r.ID = id
	}
	now := s.now()
	r.Status = StatusPending
	r.CreatedAt = now
	r.UpdatedAt = now
	if r.ExpiresAt.IsZero() {
		r.ExpiresAt = now.Add(72 * time.Hour)
	}
	rcpt, _ := json.Marshal(r.Recipients)
	mods, _ := json.Marshal(r.Moderators)
	_, err := s.db.Exec(`INSERT INTO moderation_holds
		(id, status, rule_id, sender, recipients, subject, moderators, raw_bytes, comment, expires_at, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		r.ID, string(r.Status), r.RuleID, r.Sender, string(rcpt), r.Subject, string(mods), r.Raw, r.Comment,
		r.ExpiresAt.UTC(), r.CreatedAt.UTC(), r.UpdatedAt.UTC())
	if err != nil {
		return Record{}, err
	}
	return r, nil
}

func (s *PGStore) Get(id string) (Record, bool) {
	r, err := s.scanOne(`SELECT id, status, rule_id, sender, recipients, subject, moderators, raw_bytes, comment, expires_at, created_at, updated_at
		FROM moderation_holds WHERE id=$1`, id)
	if err != nil {
		return Record{}, false
	}
	return r, true
}

func (s *PGStore) Approve(id, moderator string) (Record, error) {
	return s.act(id, moderator, StatusApproved, "")
}

func (s *PGStore) Reject(id, moderator, comment string) (Record, error) {
	if comment == "" {
		return Record{}, fmt.Errorf("reject comment required")
	}
	return s.act(id, moderator, StatusRejected, comment)
}

func (s *PGStore) act(id, moderator string, st Status, comment string) (Record, error) {
	r, ok := s.Get(id)
	if !ok {
		return Record{}, fmt.Errorf("hold %s not found", id)
	}
	if r.Status != StatusPending {
		return Record{}, fmt.Errorf("hold %s status %s", id, r.Status)
	}
	if !isModerator(r.Moderators, moderator) && moderator != "admin" {
		return Record{}, fmt.Errorf("moderator %s not allowed", moderator)
	}
	now := s.now().UTC()
	_, err := s.db.Exec(`UPDATE moderation_holds SET status=$1, comment=$2, updated_at=$3 WHERE id=$4`,
		string(st), comment, now, id)
	if err != nil {
		return Record{}, err
	}
	r.Status = st
	r.Comment = comment
	r.ConsumedBy = moderator
	r.UpdatedAt = now
	return r, nil
}

func (s *PGStore) ExpirePending(autoApprove bool) []Record {
	now := s.now().UTC()
	rows, err := s.db.Query(`SELECT id FROM moderation_holds WHERE status='pending' AND expires_at < $1`, now)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	st := StatusExpired
	comment := "ttl expired"
	if autoApprove {
		st = StatusApproved
		comment = ""
	}
	var out []Record
	for _, id := range ids {
		_, _ = s.db.Exec(`UPDATE moderation_holds SET status=$1, comment=$2, updated_at=$3 WHERE id=$4`,
			string(st), comment, now, id)
		if r, ok := s.Get(id); ok {
			out = append(out, r)
		}
	}
	return out
}

func (s *PGStore) ListPending() []Record {
	rows, err := s.db.Query(`SELECT id, status, rule_id, sender, recipients, subject, moderators, raw_bytes, comment, expires_at, created_at, updated_at
		FROM moderation_holds WHERE status='pending' ORDER BY created_at`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []Record
	for rows.Next() {
		r, err := scanRow(rows)
		if err == nil {
			out = append(out, r)
		}
	}
	return out
}

type rowScanner interface {
	Scan(dest ...any) error
}

func (s *PGStore) scanOne(q string, args ...any) (Record, error) {
	row := s.db.QueryRow(q, args...)
	return scanRow(row)
}

func scanRow(row rowScanner) (Record, error) {
	var r Record
	var status, rcpt, mods string
	var raw []byte
	if err := row.Scan(&r.ID, &status, &r.RuleID, &r.Sender, &rcpt, &r.Subject, &mods, &raw, &r.Comment, &r.ExpiresAt, &r.CreatedAt, &r.UpdatedAt); err != nil {
		return Record{}, err
	}
	r.Status = Status(status)
	r.Raw = raw
	_ = json.Unmarshal([]byte(rcpt), &r.Recipients)
	_ = json.Unmarshal([]byte(mods), &r.Moderators)
	return r, nil
}

// OpenRepositoryFromEnv — PG или memory.
func OpenRepositoryFromEnv() (Repository, error) {
	pg, err := OpenPGFromEnv()
	if err != nil {
		return nil, err
	}
	if pg != nil {
		return pg, nil
	}
	return NewStore(), nil
}