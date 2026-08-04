package store

import (
	"context"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// OpenPGFromEnv returns a pool when ERA_CHAT_DATABASE_URL or ERA_COMMS_DATABASE_URL is set.
func OpenPGFromEnv() *pgxpool.Pool {
	dsn := os.Getenv("ERA_CHAT_DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("ERA_COMMS_DATABASE_URL")
	}
	if dsn == "" {
		return nil
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil
	}
	return pool
}

// NewFromEnv — PG when DSN reachable (DDL 005_chat.sql); else JSON/memory Store.
func NewFromEnv() *Store {
	if pool := OpenPGFromEnv(); pool != nil {
		s := &Store{rooms: make(map[string]Room), pg: pool}
		s.loadFromPG()
		return s
	}
	return New()
}

func (s *Store) loadFromPG() {
	if s.pg == nil {
		return
	}
	rows, err := s.pg.Query(context.Background(), `SELECT id, tenant_id, name, created_at FROM chat_rooms`)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var r Room
		if err := rows.Scan(&r.ID, &r.TenantID, &r.Name, &r.Created); err != nil {
			continue
		}
		s.rooms[r.ID] = r
	}
	mrows, err := s.pg.Query(context.Background(), `SELECT id, room_id, tenant_id, sender, body, sent_at FROM chat_messages ORDER BY sent_at`)
	if err != nil {
		return
	}
	defer mrows.Close()
	for mrows.Next() {
		var m Message
		if err := mrows.Scan(&m.ID, &m.RoomID, &m.TenantID, &m.Sender, &m.Body, &m.SentAt); err != nil {
			continue
		}
		s.messages = append(s.messages, m)
		s.seq++
	}
}

func (s *Store) createRoomPG(tenantID, name string) Room {
	s.seq++
	id := "!" + name + ":era"
	if name == "" {
		id = "!room-" + time.Now().UTC().Format("150405") + "-" + itoa(s.seq) + ":era"
	}
	r := Room{ID: id, TenantID: tenantID, Name: name, Created: time.Now().UTC()}
	_, _ = s.pg.Exec(context.Background(),
		`INSERT INTO chat_rooms (id, tenant_id, name, created_at) VALUES ($1,$2,$3,$4) ON CONFLICT (id) DO NOTHING`,
		r.ID, r.TenantID, r.Name, r.Created)
	s.rooms[id] = r
	return r
}

func (s *Store) addMessagePG(tenantID, roomID, sender, body string) (Message, bool) {
	r, ok := s.rooms[roomID]
	if !ok || r.TenantID != tenantID {
		var tid, name string
		var created time.Time
		err := s.pg.QueryRow(context.Background(),
			`SELECT tenant_id, name, created_at FROM chat_rooms WHERE id=$1`, roomID).Scan(&tid, &name, &created)
		if err != nil || tid != tenantID {
			return Message{}, false
		}
		s.rooms[roomID] = Room{ID: roomID, TenantID: tid, Name: name, Created: created}
	}
	s.seq++
	m := Message{
		ID:       "$" + itoa(s.seq) + ":era",
		RoomID:   roomID,
		TenantID: tenantID,
		Sender:   sender,
		Body:     body,
		SentAt:   time.Now().UTC(),
	}
	_, err := s.pg.Exec(context.Background(),
		`INSERT INTO chat_messages (id, room_id, tenant_id, sender, body, sent_at) VALUES ($1,$2,$3,$4,$5,$6)`,
		m.ID, m.RoomID, m.TenantID, m.Sender, m.Body, m.SentAt)
	if err != nil {
		return Message{}, false
	}
	s.messages = append(s.messages, m)
	return m, true
}
