package store

import "time"

// CaseNote — заметка аналитика (F-GA-7).
type CaseNote struct {
	ID        string    `json:"id"`
	CaseID    string    `json:"case_id"`
	Author    string    `json:"author"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

// TimelineEvent — событие в timeline кейса.
type TimelineEvent struct {
	ID        string    `json:"id"`
	CaseID    string    `json:"case_id"`
	Kind      string    `json:"kind"`
	Actor     string    `json:"actor"`
	Detail    string    `json:"detail"`
	CreatedAt time.Time `json:"created_at"`
}

// AuditEntry — audit trail control-plane (GA-2 prep).
type AuditEntry struct {
	ID        string    `json:"id"`
	Action    string    `json:"action"`
	Actor     string    `json:"actor"`
	Target    string    `json:"target"`
	Detail    string    `json:"detail"`
	CreatedAt time.Time `json:"created_at"`
}

// CaseDetail — case + notes + timeline.
type CaseDetail struct {
	*Case
	Notes    []*CaseNote    `json:"notes"`
	Timeline []*TimelineEvent `json:"timeline"`
}
