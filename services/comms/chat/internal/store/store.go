package store

import (
	"sync"
	"time"
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

type Store struct {
	mu       sync.RWMutex
	rooms    map[string]Room
	messages []Message
	seq      int
}

func New() *Store {
	return &Store{
		rooms: make(map[string]Room),
	}
}

func (s *Store) CreateRoom(tenantID, name string) Room {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	id := "room-" + time.Now().UTC().Format("150405") + "-" + itoa(s.seq)
	r := Room{ID: id, TenantID: tenantID, Name: name, Created: time.Now().UTC()}
	s.rooms[id] = r
	return r
}

func (s *Store) AddMessage(tenantID, roomID, sender, body string) (Message, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.rooms[roomID]
	if !ok || r.TenantID != tenantID {
		return Message{}, false
	}
	s.seq++
	m := Message{
		ID:       "msg-" + itoa(s.seq),
		RoomID:   roomID,
		TenantID: tenantID,
		Sender:   sender,
		Body:     body,
		SentAt:   time.Now().UTC(),
	}
	s.messages = append(s.messages, m)
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
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
