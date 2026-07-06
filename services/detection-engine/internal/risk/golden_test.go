package risk

import (
	"testing"
	"time"
)

func TestCaseEscalationGolden(t *testing.T) {
	s := New(15 * time.Minute)
	at := time.Now().UTC()
	d := DecideEscalation("node-1", s, at, []string{"critical", "high"})
	if !d.Escalate || d.Score < 30 {
		t.Fatalf("expected escalation, got %+v", d)
	}
	d2 := DecideEscalation("node-2", s, at, []string{"low"})
	if d2.Escalate {
		t.Fatalf("low severity should not escalate: %+v", d2)
	}
}
