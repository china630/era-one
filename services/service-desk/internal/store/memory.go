package store

import "sync"

type memoryStore struct {
	mu        sync.RWMutex
	incidents []*Incident
	requests  []*ServiceRequest
	problems  []*Problem
	changes   []*Change
}

func NewMemory() Repository {
	return &memoryStore{}
}

func (m *memoryStore) CreateIncident(i *Incident) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := nowUTC()
	i.CreatedAt = now
	i.UpdatedAt = now
	if i.Status == "" {
		i.Status = StatusNew
	}
	m.incidents = append(m.incidents, i)
}

func (m *memoryStore) GetIncident(id string) (*Incident, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, x := range m.incidents {
		if x.ID == id {
			return x, true
		}
	}
	return nil, false
}

func (m *memoryStore) UpdateIncident(id string, fn func(*Incident)) (*Incident, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, x := range m.incidents {
		if x.ID == id {
			fn(x)
			x.UpdatedAt = nowUTC()
			return x, true
		}
	}
	return nil, false
}

func (m *memoryStore) ListIncidents() []*Incident {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*Incident, len(m.incidents))
	copy(out, m.incidents)
	return out
}

func (m *memoryStore) CreateRequest(r *ServiceRequest) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := nowUTC()
	r.CreatedAt = now
	r.UpdatedAt = now
	if r.Status == "" {
		r.Status = StatusNew
	}
	m.requests = append(m.requests, r)
}

func (m *memoryStore) ListRequests() []*ServiceRequest {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*ServiceRequest, len(m.requests))
	copy(out, m.requests)
	return out
}

func (m *memoryStore) CreateProblem(p *Problem) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p.CreatedAt = nowUTC()
	if p.Status == "" {
		p.Status = StatusNew
	}
	m.problems = append(m.problems, p)
}

func (m *memoryStore) ListProblems() []*Problem {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]*Problem(nil), m.problems...)
}

func (m *memoryStore) CreateChange(c *Change) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c.CreatedAt = nowUTC()
	if c.Status == "" {
		c.Status = StatusNew
	}
	m.changes = append(m.changes, c)
}

func (m *memoryStore) ListChanges() []*Change {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]*Change(nil), m.changes...)
}
