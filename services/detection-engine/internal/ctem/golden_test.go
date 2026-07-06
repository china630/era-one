package ctem

import "testing"

func TestCTEMScoreGolden(t *testing.T) {
	if got := Score(3, 0.5); got != 44.0 {
		t.Fatalf("score=%v want 44", got)
	}
	if Score(0, 0) != 0 {
		t.Fatal("zero inputs")
	}
}
