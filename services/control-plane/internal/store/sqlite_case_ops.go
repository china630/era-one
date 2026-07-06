package store

import "time"

func (s *sqliteStore) GetCaseDetail(id string) (*CaseDetail, bool) {
	c, ok := s.GetCase(id)
	if !ok {
		return nil, false
	}
	notes := s.listNotes(id)
	timeline := s.listTimeline(id)
	return &CaseDetail{Case: c, Notes: notes, Timeline: timeline}, true
}

func (s *sqliteStore) AddCaseNote(caseID, author, body string) (*CaseNote, bool) {
	if _, ok := s.GetCase(caseID); !ok {
		return nil, false
	}
	n := newNote(caseID, author, body)
	_, _ = s.db.Exec(`INSERT INTO case_notes(id,case_id,author,body,created_at) VALUES(?,?,?,?,?)`,
		n.ID, n.CaseID, n.Author, n.Body, n.CreatedAt.Format(time.RFC3339Nano))
	return n, true
}

func (s *sqliteStore) AddTimeline(caseID, kind, actor, detail string) (*TimelineEvent, bool) {
	if _, ok := s.GetCase(caseID); !ok {
		return nil, false
	}
	ev := newTimeline(caseID, kind, actor, detail)
	_, _ = s.db.Exec(`INSERT INTO case_timeline(id,case_id,kind,actor,detail,created_at) VALUES(?,?,?,?,?,?)`,
		ev.ID, ev.CaseID, ev.Kind, ev.Actor, ev.Detail, ev.CreatedAt.Format(time.RFC3339Nano))
	return ev, true
}

func (s *sqliteStore) RecordAudit(action, actor, target, detail string) {
	a := newAudit(action, actor, target, detail)
	_, _ = s.db.Exec(`INSERT INTO audit_log(id,action,actor,target,detail,created_at) VALUES(?,?,?,?,?,?)`,
		a.ID, a.Action, a.Actor, a.Target, a.Detail, a.CreatedAt.Format(time.RFC3339Nano))
}

func (s *sqliteStore) ListAudit(limit int) []*AuditEntry {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.Query(`SELECT id,action,actor,target,detail,created_at FROM audit_log ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil
	}
	defer rows.Close()
	return scanAudit(rows)
}

func (s *sqliteStore) listNotes(caseID string) []*CaseNote {
	rows, err := s.db.Query(`SELECT id,case_id,author,body,created_at FROM case_notes WHERE case_id=? ORDER BY created_at`, caseID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []*CaseNote
	for rows.Next() {
		var n CaseNote
		var ts string
		if err := rows.Scan(&n.ID, &n.CaseID, &n.Author, &n.Body, &ts); err != nil {
			continue
		}
		n.CreatedAt, _ = time.Parse(time.RFC3339Nano, ts)
		out = append(out, &n)
	}
	return out
}

func (s *sqliteStore) listTimeline(caseID string) []*TimelineEvent {
	rows, err := s.db.Query(`SELECT id,case_id,kind,actor,detail,created_at FROM case_timeline WHERE case_id=? ORDER BY created_at`, caseID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []*TimelineEvent
	for rows.Next() {
		var ev TimelineEvent
		var ts string
		if err := rows.Scan(&ev.ID, &ev.CaseID, &ev.Kind, &ev.Actor, &ev.Detail, &ts); err != nil {
			continue
		}
		ev.CreatedAt, _ = time.Parse(time.RFC3339Nano, ts)
		out = append(out, &ev)
	}
	return out
}

func scanAudit(rows interface {
	Next() bool
	Scan(dest ...any) error
}) []*AuditEntry {
	var out []*AuditEntry
	for rows.Next() {
		var a AuditEntry
		var ts string
		if err := rows.Scan(&a.ID, &a.Action, &a.Actor, &a.Target, &a.Detail, &ts); err != nil {
			continue
		}
		a.CreatedAt, _ = time.Parse(time.RFC3339Nano, ts)
		out = append(out, &a)
	}
	return out
}
