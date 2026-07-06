package processor

import (
	"context"
	"sync"
	"testing"
	"time"

	erav1 "era/contracts/gen/era/v1"
	"era/services/detection-engine/internal/chwriter"
	"era/services/detection-engine/internal/risk"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type memWriter struct {
	mu   sync.Mutex
	rows []chwriter.DetectionRow
}

func (m *memWriter) InsertDetection(_ context.Context, d chwriter.DetectionRow) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rows = append(m.rows, d)
	return nil
}

func (m *memWriter) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.rows)
}

func TestRiskDedupSuppressesRepeatDetection(t *testing.T) {
	w := &memWriter{}
	p := &Processor{
		NDR:        nil,
		Risk:       risk.New(15 * time.Minute),
		Detections: w,
	}
	obs := time.Now().UTC()
	node := "node-risk-1"
	env := &erav1.Envelope{
		ObservedAt: timestamppb.New(obs),
		Source:     &erav1.Source{NodeId: node, TenantId: "t1"},
		Payload:    &erav1.Envelope_Raw{},
	}
	for i := 0; i < 3; i++ {
		p.emit(context.Background(), env, "era-test-rule", "test rule", "high", "test", "ev-1", obs.Add(time.Duration(i)*time.Second), node)
	}
	if got := w.count(); got != 1 {
		t.Fatalf("dedup: rows=%d want 1", got)
	}
}

func TestRiskEscalationOnHighScore(t *testing.T) {
	w := &memWriter{}
	p := &Processor{
		Risk:       risk.New(15 * time.Minute),
		Detections: w,
	}
	obs := time.Now().UTC()
	node := "node-esc"
	env := &erav1.Envelope{
		ObservedAt: timestamppb.New(obs),
		Source:     &erav1.Source{NodeId: node},
		Payload:    &erav1.Envelope_Raw{},
	}
	rules := []struct{ id, sev string }{
		{"r-critical", "critical"},
		{"r-high-1", "high"},
		{"r-high-2", "high"},
		{"r-high-3", "high"},
	}
	for i, r := range rules {
		p.emit(context.Background(), env, r.id, "rule", r.sev, "test", "ev", obs.Add(time.Duration(i)*time.Minute), node)
	}
	if w.count() < 4 {
		t.Fatalf("expected 4 distinct rules, got %d", w.count())
	}
}
