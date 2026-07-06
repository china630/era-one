package store

import "github.com/google/uuid"

// CaseOps — notes, timeline, audit (GA-1 S5-11).
type CaseOps interface {
	GetCaseDetail(id string) (*CaseDetail, bool)
	AddCaseNote(caseID, author, body string) (*CaseNote, bool)
	AddTimeline(caseID, kind, actor, detail string) (*TimelineEvent, bool)
	RecordAudit(action, actor, target, detail string)
	ListAudit(limit int) []*AuditEntry
}

func newNote(caseID, author, body string) *CaseNote {
	return &CaseNote{
		ID: uuid.NewString(), CaseID: caseID, Author: author, Body: body, CreatedAt: nowUTC(),
	}
}

func newTimeline(caseID, kind, actor, detail string) *TimelineEvent {
	return &TimelineEvent{
		ID: uuid.NewString(), CaseID: caseID, Kind: kind, Actor: actor, Detail: detail, CreatedAt: nowUTC(),
	}
}

func newAudit(action, actor, target, detail string) *AuditEntry {
	return &AuditEntry{
		ID: uuid.NewString(), Action: action, Actor: actor, Target: target, Detail: detail, CreatedAt: nowUTC(),
	}
}
