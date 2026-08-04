package investigate

import (
	"sync"
	"time"
)

// AuditEntry is an immutable investigation audit row (ADR-0023 AI-4).
type AuditEntry struct {
	At           time.Time `json:"at"`
	Actor        string    `json:"actor"`
	CaseID       string    `json:"case_id,omitempty"`
	NodeID       string    `json:"node_id"`
	DetectionID  string    `json:"detection_id,omitempty"`
	Verdict      string    `json:"verdict"`
	ModelVersion string    `json:"model_version"`
	PromptHash   string    `json:"prompt_hash"`
	CustodyHash  string    `json:"custody_root_hash,omitempty"`
}

// AuditLog is append-only in-memory (MVP).
type AuditLog struct {
	mu   sync.Mutex
	rows []AuditEntry
}

func NewAuditLog() *AuditLog { return &AuditLog{} }

func (a *AuditLog) Append(e AuditEntry) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if e.At.IsZero() {
		e.At = time.Now().UTC()
	}
	a.rows = append(a.rows, e)
}

func (a *AuditLog) List() []AuditEntry {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]AuditEntry, len(a.rows))
	copy(out, a.rows)
	return out
}
