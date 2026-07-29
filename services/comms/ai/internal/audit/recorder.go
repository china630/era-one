package audit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync"
)

type Event struct {
	TenantID      string
	MailboxID     string
	InferenceType string
	Model         string
	RiskScore     int
	LatencyMs     int64
	RequestID     string
	BodyHash      string
}

type Recorder struct {
	mu     sync.Mutex
	events []Event
	CH     CHWriter
}

type CHWriter interface {
	RecordAIInference(ctx context.Context, tenantID, mailboxID, inferenceType, model string, riskScore int, latencyMs int64, requestID, bodyHash string) error
}

func NewRecorder(ch CHWriter) *Recorder {
	return &Recorder{CH: ch}
}

func (r *Recorder) Record(ctx context.Context, ev Event) {
	r.mu.Lock()
	r.events = append(r.events, ev)
	r.mu.Unlock()
	if r.CH != nil {
		_ = r.CH.RecordAIInference(ctx, ev.TenantID, ev.MailboxID, ev.InferenceType, ev.Model, ev.RiskScore, ev.LatencyMs, ev.RequestID, ev.BodyHash)
	}
}

func (r *Recorder) Count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.events)
}

func BodyHash(body string) string {
	sum := sha256.Sum256([]byte(body))
	return hex.EncodeToString(sum[:8])
}
