package bas

import (
	"context"
	"testing"

	"era/services/ctem/internal/risk"
)

func TestBASFeedbackRaisesRiskScore(t *testing.T) {
	var r Runner
	if err := r.SimulateLateral(context.Background(), "10.0.0.99"); err != nil {
		t.Fatal(err)
	}
	before := risk.ComputeRisk(risk.Input{
		NodeID: "10.0.0.99", Platform: "windows",
		CVECounts: map[string]int{"critical": 1},
	})
	after := risk.ComputeRisk(risk.Input{
		NodeID: "10.0.0.99", Platform: "windows",
		CVECounts: map[string]int{"critical": 1},
		BASDetected: true,
	})
	if after.Total-before.Total < 20 {
		t.Fatalf("BAS feedback loop: before=%v after=%v", before.Total, after.Total)
	}
}
