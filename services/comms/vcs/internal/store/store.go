package store

import (
	"sync"
	"time"
)

type Room struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Name      string    `json:"name"`
	LKRoom    string    `json:"lk_room"`
	CreatedAt time.Time `json:"created_at"`
}

type Store struct {
	mu    sync.RWMutex
	rooms map[string]Room
	seq   int
}

func New() *Store {
	return &Store{rooms: make(map[string]Room)}
}

func (s *Store) Put(r Room) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rooms[r.ID] = r
}

func (s *Store) Get(id string) (Room, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.rooms[id]
	return r, ok
}

func (s *Store) NextID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	return "vcs-" + itoa(s.seq)
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
