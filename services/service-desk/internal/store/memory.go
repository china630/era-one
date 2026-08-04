package store

import "sync"

type memoryStore struct {
	mu        sync.RWMutex
	incidents []*Incident
	requests  []*ServiceRequest
	problems  []*Problem
	changes   []*Change
	comments  []*Comment
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
	if r.SLAStatus == "" {
		r.SLAStatus = "none"
	}
	m.requests = append(m.requests, r)
}

func (m *memoryStore) GetRequest(id string) (*ServiceRequest, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, x := range m.requests {
		if x.ID == id {
			return x, true
		}
	}
	return nil, false
}

func (m *memoryStore) UpdateRequest(id string, fn func(*ServiceRequest)) (*ServiceRequest, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, x := range m.requests {
		if x.ID == id {
			fn(x)
			x.UpdatedAt = nowUTC()
			return x, true
		}
	}
	return nil, false
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
	now := nowUTC()
	p.CreatedAt = now
	p.UpdatedAt = now
	if p.Status == "" {
		p.Status = StatusNew
	}
	if p.SLAStatus == "" {
		p.SLAStatus = "none"
	}
	m.problems = append(m.problems, p)
}

func (m *memoryStore) GetProblem(id string) (*Problem, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, x := range m.problems {
		if x.ID == id {
			return x, true
		}
	}
	return nil, false
}

func (m *memoryStore) UpdateProblem(id string, fn func(*Problem)) (*Problem, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, x := range m.problems {
		if x.ID == id {
			fn(x)
			x.UpdatedAt = nowUTC()
			return x, true
		}
	}
	return nil, false
}

func (m *memoryStore) ListProblems() []*Problem {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]*Problem(nil), m.problems...)
}

func (m *memoryStore) CreateChange(c *Change) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := nowUTC()
	c.CreatedAt = now
	c.UpdatedAt = now
	if c.Status == "" {
		c.Status = StatusNew
	}
	if c.SLAStatus == "" {
		c.SLAStatus = "none"
	}
	m.changes = append(m.changes, c)
}

func (m *memoryStore) GetChange(id string) (*Change, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, x := range m.changes {
		if x.ID == id {
			return x, true
		}
	}
	return nil, false
}

func (m *memoryStore) UpdateChange(id string, fn func(*Change)) (*Change, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, x := range m.changes {
		if x.ID == id {
			fn(x)
			x.UpdatedAt = nowUTC()
			return x, true
		}
	}
	return nil, false
}

func (m *memoryStore) ListChanges() []*Change {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]*Change(nil), m.changes...)
}

func (m *memoryStore) AddComment(c *Comment) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c.CreatedAt = nowUTC()
	m.comments = append(m.comments, c)
}

func (m *memoryStore) ListComments(kind TicketKind, ticketID string) []*Comment {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*Comment
	for _, c := range m.comments {
		if c.Kind == kind && c.TicketID == ticketID {
			out = append(out, c)
		}
	}
	return out
}
