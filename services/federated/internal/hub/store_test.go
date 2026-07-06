package hub

import (
	"path/filepath"
	"testing"
)

func TestPersistentHubSQLite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hub.db")
	store, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	h := NewPersistent(2.0, store)
	if err := h.Submit(GradientSubmission{ZoneID: "zone-a", Vector: []float64{0.2, 0.4}, SampleCount: 100}); err != nil {
		t.Fatal(err)
	}
	if err := h.Submit(GradientSubmission{ZoneID: "zone-b", Vector: []float64{0.6, 0.8}, SampleCount: 200}); err != nil {
		t.Fatal(err)
	}
	model, round := h.Aggregate()
	if round != 1 || len(model) != 2 {
		t.Fatalf("round=%d model=%v", round, model)
	}

	h2 := NewPersistent(2.0, store)
	model2, round2 := h2.GlobalModel()
	if round2 != 1 || len(model2) != 2 {
		t.Fatalf("reload round=%d model=%v", round2, model2)
	}
}
