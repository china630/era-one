// Package hub — in-memory TAXII collections (air-gap on-prem).
package hub

import (
	"sync"
	"time"

	"era/services/national-hub/internal/dp"
)

const DefaultCollection = "era-national-threats"

type Object struct {
	ID        string    `json:"id"`
	OrgID     string    `json:"org_id"`
	Data      []byte    `json:"-"`
	RawJSON   string    `json:"data"`
	CreatedAt time.Time `json:"created_at"`
}

type Store struct {
	mu          sync.RWMutex
	objects     map[string][]Object
	subscribers map[string][]string
}

func NewStore() *Store {
	return &Store{
		objects:     make(map[string][]Object),
		subscribers: make(map[string][]string),
	}
}

func (s *Store) Publish(collection, orgID, objID string, data []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.objects[collection] = append(s.objects[collection], Object{
		ID: objID, OrgID: orgID, Data: data, RawJSON: string(data),
		CreatedAt: time.Now().UTC(),
	})
}

func (s *Store) Poll(collection string) []Object {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Object, len(s.objects[collection]))
	copy(out, s.objects[collection])
	return out
}

func (s *Store) Subscribe(orgID, collection string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subscribers[collection] = append(s.subscribers[collection], orgID)
}

func (s *Store) SubscriberCount(collection string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.subscribers[collection])
}

func (s *Store) PublishCount(collection string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.objects[collection])
}

func (s *Store) NoisyPublishCount(collection string, epsilon float64) float64 {
	return dp.NoisyCount(s.PublishCount(collection), epsilon)
}
