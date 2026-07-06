package store

func (m *memoryStore) GetCaseDetail(id string) (*CaseDetail, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.cases[id]
	if !ok {
		return nil, false
	}
	return &CaseDetail{
		Case:     c,
		Notes:    append([]*CaseNote{}, m.notes[id]...),
		Timeline: append([]*TimelineEvent{}, m.timeline[id]...),
	}, true
}

func (m *memoryStore) AddCaseNote(caseID, author, body string) (*CaseNote, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.cases[caseID]; !ok {
		return nil, false
	}
	n := newNote(caseID, author, body)
	m.notes[caseID] = append(m.notes[caseID], n)
	return n, true
}

func (m *memoryStore) AddTimeline(caseID, kind, actor, detail string) (*TimelineEvent, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.cases[caseID]; !ok {
		return nil, false
	}
	ev := newTimeline(caseID, kind, actor, detail)
	m.timeline[caseID] = append(m.timeline[caseID], ev)
	return ev, true
}

func (m *memoryStore) RecordAudit(action, actor, target, detail string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.audit = append(m.audit, newAudit(action, actor, target, detail))
}

func (m *memoryStore) ListAudit(limit int) []*AuditEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if limit <= 0 || limit > len(m.audit) {
		limit = len(m.audit)
	}
	start := len(m.audit) - limit
	if start < 0 {
		start = 0
	}
	out := make([]*AuditEntry, limit)
	copy(out, m.audit[start:])
	return out
}
