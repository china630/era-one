package store

import (
	"database/sql"
)

func (s *postgresStore) GetCaseDetail(id string) (*CaseDetail, bool) {
	c, ok := s.GetCase(id)
	if !ok {
		return nil, false
	}
	return &CaseDetail{Case: c, Notes: s.listNotes(id), Timeline: s.listTimeline(id)}, true
}

func (s *postgresStore) AddCaseNote(caseID, author, body string) (*CaseNote, bool) {
	if _, ok := s.GetCase(caseID); !ok {
		return nil, false
	}
	n := newNote(caseID, author, body)
	_, _ = s.db.Exec(`INSERT INTO case_notes(id,case_id,author,body,created_at) VALUES($1,$2,$3,$4,$5)`,
		n.ID, n.CaseID, n.Author, n.Body, n.CreatedAt)
	return n, true
}

func (s *postgresStore) AddTimeline(caseID, kind, actor, detail string) (*TimelineEvent, bool) {
	if _, ok := s.GetCase(caseID); !ok {
		return nil, false
	}
	ev := newTimeline(caseID, kind, actor, detail)
	_, _ = s.db.Exec(`INSERT INTO case_timeline(id,case_id,kind,actor,detail,created_at) VALUES($1,$2,$3,$4,$5,$6)`,
		ev.ID, ev.CaseID, ev.Kind, ev.Actor, ev.Detail, ev.CreatedAt)
	return ev, true
}

func (s *postgresStore) RecordAudit(action, actor, target, detail string) {
	a := newAudit(action, actor, target, detail)
	_, _ = s.db.Exec(`INSERT INTO audit_log(id,action,actor,target,detail,created_at) VALUES($1,$2,$3,$4,$5,$6)`,
		a.ID, a.Action, a.Actor, a.Target, a.Detail, a.CreatedAt)
}

func (s *postgresStore) ListAudit(limit int) []*AuditEntry {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.Query(`SELECT id,action,actor,target,detail,created_at FROM audit_log ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil
	}
	defer rows.Close()
	return scanAuditPG(rows)
}

func (s *postgresStore) listNotes(caseID string) []*CaseNote {
	rows, err := s.db.Query(`SELECT id,case_id,author,body,created_at FROM case_notes WHERE case_id=$1 ORDER BY created_at`, caseID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []*CaseNote
	for rows.Next() {
		var n CaseNote
		if err := rows.Scan(&n.ID, &n.CaseID, &n.Author, &n.Body, &n.CreatedAt); err != nil {
			continue
		}
		out = append(out, &n)
	}
	return out
}

func (s *postgresStore) listTimeline(caseID string) []*TimelineEvent {
	rows, err := s.db.Query(`SELECT id,case_id,kind,actor,detail,created_at FROM case_timeline WHERE case_id=$1 ORDER BY created_at`, caseID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []*TimelineEvent
	for rows.Next() {
		var ev TimelineEvent
		if err := rows.Scan(&ev.ID, &ev.CaseID, &ev.Kind, &ev.Actor, &ev.Detail, &ev.CreatedAt); err != nil {
			continue
		}
		out = append(out, &ev)
	}
	return out
}

func scanAuditPG(rows *sql.Rows) []*AuditEntry {
	var out []*AuditEntry
	for rows.Next() {
		var a AuditEntry
		if err := rows.Scan(&a.ID, &a.Action, &a.Actor, &a.Target, &a.Detail, &a.CreatedAt); err != nil {
			continue
		}
		out = append(out, &a)
	}
	return out
}
