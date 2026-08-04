package processor

import (
	"context"
	"testing"
	"time"

	erav1 "era/contracts/gen/era/v1"
	"era/services/detection-engine/internal/suppress"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestSuppressBlocksEmit(t *testing.T) {
	w := &memWriter{}
	sc := suppress.New(nil)
	sc.Set([]suppress.Entry{{RuleID: "era-fp", NodeID: "n1"}})
	p := &Processor{Detections: w, Suppress: sc}
	obs := time.Now().UTC()
	env := &erav1.Envelope{
		ObservedAt: timestamppb.New(obs),
		Source:     &erav1.Source{NodeId: "n1", TenantId: "t1"},
	}
	p.emit(context.Background(), env, "era-fp", "fp rule", "high", "test", "e1", obs, "n1", nil)
	if w.count() != 0 {
		t.Fatalf("expected suppress, got %d", w.count())
	}
	p.emit(context.Background(), env, "era-other", "other", "high", "test", "e2", obs, "n1", nil)
	if w.count() != 1 {
		t.Fatalf("expected other emit, got %d", w.count())
	}
}
