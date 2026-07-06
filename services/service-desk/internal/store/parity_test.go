package store

import (
	"path/filepath"
	"testing"
	"time"
)

func runParityScenario(t *testing.T, st Repository) {
	t.Helper()
	past := time.Now().UTC().Add(-time.Hour)
	future := time.Now().UTC().Add(time.Hour)
	st.CreateIncident(&Incident{ID: "i1", Title: "a", Status: StatusNew, SLADueAt: &future})
	st.CreateIncident(&Incident{ID: "i2", Title: "b", Status: StatusInProgress, SLADueAt: &past})
	if len(st.ListIncidents()) != 2 {
		t.Fatalf("incidents: %d", len(st.ListIncidents()))
	}
	st.CreateRequest(&ServiceRequest{ID: "r1", Title: "req", Requester: "u1"})
	if len(st.ListRequests()) != 1 {
		t.Fatal("requests")
	}
}

func TestMemorySQLiteParity(t *testing.T) {
	runParityScenario(t, NewMemory())
	dir := t.TempDir()
	st, err := NewSQLite(filepath.Join(dir, "desk.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if c, ok := st.(CloseableRepository); ok {
			_ = c.Close()
		}
	}()
	runParityScenario(t, st)
}
