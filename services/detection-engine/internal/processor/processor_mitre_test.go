package processor

import (
	"context"
	"testing"
	"time"

	erav1 "era/contracts/gen/era/v1"
	"era/services/detection-engine/internal/sigma"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestSigmaEmitMitreTechniques(t *testing.T) {
	w := &memWriter{}
	rule := &sigma.Rule{
		ID:    "era-test-timestomp",
		Title: "Timestomp",
		Level: "medium",
		Tags:  []string{"attack.T1099"},
		Logsource: map[string]string{"category": "process"},
		Detection: map[string]any{
			"selection": map[string]any{"CommandLine|contains": "timestomp"},
			"condition": "selection",
		},
	}
	p := &Processor{Rules: []*sigma.Rule{rule}, Detections: w, Corr: nil}
	// Corr.Observe will panic if Corr is nil - need correlator
	p = New([]*sigma.Rule{rule}, w, nil, nil, nil)
	obs := time.Now().UTC()
	env := &erav1.Envelope{
		Category:   erav1.EventCategory_EVENT_CATEGORY_PROCESS,
		ObservedAt: timestamppb.New(obs),
		Source:     &erav1.Source{NodeId: "n1", TenantId: "t1"},
		Payload: &erav1.Envelope_Process{
			Process: &erav1.ProcessEvent{CommandLine: "timestomp -r file.exe"},
		},
	}
	p.Handle(context.Background(), env)
	if w.count() != 1 {
		t.Fatalf("rows=%d", w.count())
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	tech := w.rows[0].MitreTechniques
	if len(tech) != 1 || tech[0] != "T1099" {
		t.Fatalf("mitre=%v want [T1099]", tech)
	}
}
