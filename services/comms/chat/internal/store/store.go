package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Room struct {
	ID       string    `json:"id"`
	TenantID string    `json:"tenant_id"`
	Name     string    `json:"name"`
	Created  time.Time `json:"created_at"`
}

type Message struct {
	ID       string    `json:"id"`
	RoomID   string    `json:"room_id"`
	TenantID string    `json:"tenant_id"`
	Sender   string    `json:"sender"`
	Body     string    `json:"body"`
	SentAt   time.Time `json:"sent_at"`
}

type snapshot struct {
	Rooms    map[string]Room `json:"rooms"`
	Messages []Message       `json:"messages"`
	Seq      int             `json:"seq"`
}

// Store — in-memory with optional JSON file persistence (ERA_CHAT_DATA_DIR)
// or Postgres when ERA_CHAT_DATABASE_URL / ERA_COMMS_DATABASE_URL is set (L-3).
type Store struct {
	mu       sync.RWMutex
	rooms    map[string]Room
	messages []Message
	seq      int
	path     string
	pg       *pgxpool.Pool
}

func New() *Store {
	s := &Store{rooms: make(map[string]Room)}
	if dir := os.Getenv("ERA_CHAT_DATA_DIR"); dir != "" {
		_ = os.MkdirAll(dir, 0o755)
		s.path = filepath.Join(dir, "chat-store.json")
		s.load()
	}
	return s
}

// Backend returns storage honesty label for /healthz (memory | json | postgres).
func (s *Store) Backend() string {
	if s == nil {
		return "memory"
	}
	if s.pg != nil {
		return "postgres"
	}
	if s.path != "" {
		return "json"
	}
	return "memory"
}

func (s *Store) load() {
	b, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	var snap snapshot
	if json.Unmarshal(b, &snap) != nil {
		return
	}
	if snap.Rooms == nil {
		snap.Rooms = map[string]Room{}
	}
	s.rooms = snap.Rooms
	s.messages = snap.Messages
	s.seq = snap.Seq
}

func (s *Store) persistLocked() {
	if s.path == "" {
		return
	}
	snap := snapshot{Rooms: s.rooms, Messages: s.messages, Seq: s.seq}
	b, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(s.path, b, 0o600)
}

func (s *Store) CreateRoom(tenantID, name string) Room {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pg != nil {
		return s.createRoomPG(tenantID, name)
	}
	s.seq++
	id := "!" + name + ":era"
	if name == "" {
		id = "!room-" + time.Now().UTC().Format("150405") + "-" + itoa(s.seq) + ":era"
	}
	r := Room{ID: id, TenantID: tenantID, Name: name, Created: time.Now().UTC()}
	s.rooms[id] = r
	s.persistLocked()
	return r
}

func (s *Store) AddMessage(tenantID, roomID, sender, body string) (Message, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pg != nil {
		return s.addMessagePG(tenantID, roomID, sender, body)
	}
	r, ok := s.rooms[roomID]
	if !ok || r.TenantID != tenantID {
		return Message{}, false
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
	s.messages = append(s.messages, m)
	s.persistLocked()
	return m, true
}

func (s *Store) ListMessages(tenantID, roomID string) []Message {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []Message
	for _, m := range s.messages {
		if m.TenantID == tenantID && m.RoomID == roomID {
			out = append(out, m)
		}
	}
	return out
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
