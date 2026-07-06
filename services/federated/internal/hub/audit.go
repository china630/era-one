package hub

import (
	"sync"
	"time"
)

// SubmissionAudit — запись аудита gradient submission (L-01).
type SubmissionAudit struct {
	ZoneID      string    `json:"zone_id"`
	VectorDim   int       `json:"vector_dim"`
	SampleCount int       `json:"sample_count"`
	At          time.Time `json:"at"`
}

// AuditLog — in-memory журнал submissions (без векторов).
type AuditLog struct {
	mu      sync.RWMutex
	entries []SubmissionAudit
}

func NewAuditLog() *AuditLog {
	return &AuditLog{}
}

func (a *AuditLog) Record(sub GradientSubmission) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.entries = append(a.entries, SubmissionAudit{
		ZoneID:      sub.ZoneID,
		VectorDim:   len(sub.Vector),
		SampleCount: sub.SampleCount,
		At:          time.Now().UTC(),
	})
}

func (a *AuditLog) Entries() []SubmissionAudit {
	a.mu.RLock()
	defer a.mu.RUnlock()
	out := make([]SubmissionAudit, len(a.entries))
	copy(out, a.entries)
	return out
}

func (a *AuditLog) Count() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.entries)
}
