package hub

import (
	"encoding/json"
	"testing"
)

func TestFederatedTwoZones(t *testing.T) {
	h := New(2.0)
	if err := h.Submit(GradientSubmission{ZoneID: "zone-a", Vector: []float64{0.2, 0.4}, SampleCount: 100}); err != nil {
		t.Fatal(err)
	}
	if err := h.Submit(GradientSubmission{ZoneID: "zone-b", Vector: []float64{0.6, 0.8}, SampleCount: 200}); err != nil {
		t.Fatal(err)
	}
	model, round := h.Aggregate()
	if round != 1 {
		t.Fatalf("round=%d", round)
	}
	if len(model) != 2 {
		t.Fatalf("model dim=%d", len(model))
	}
	if round != 1 {
		t.Fatalf("round=%d", round)
	}
}

func TestNoPIIInGradientJSON(t *testing.T) {
	body, _ := json.Marshal(GradientSubmission{
		ZoneID: "zone-a", Vector: []float64{0.1, 0.2}, SampleCount: 50,
	})
	s := string(body)
	for _, bad := range []string{"password", "alice@", "email"} {
		if contains(s, bad) {
			t.Fatalf("PII in gradient payload: %s", bad)
		}
	}
}

func contains(s, sub string) bool {
	return len(sub) > 0 && search(s, sub)
}

func search(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
