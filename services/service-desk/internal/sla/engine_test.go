package sla

import (
	"testing"
	"time"

	"era/services/service-desk/internal/store"
)

func TestSLABreachEscalatesPriority(t *testing.T) {
	st := store.NewMemory()
	past := time.Now().UTC().Add(-1 * time.Hour)
	st.CreateIncident(&store.Incident{
		ID: "inc-1", Title: "disk full", Priority: "low", Status: store.StatusNew,
		SLADueAt: &past,
	})
	eng := NewEngine(st)
	eng.Now = func() time.Time { return time.Now().UTC() }
	breached := eng.CheckBreaches()
	if len(breached) != 1 {
		t.Fatalf("breached: %v", breached)
	}
	inc, ok := st.GetIncident("inc-1")
	if !ok || !inc.SLABreached {
		t.Fatal("sla_breached not set")
	}
	if inc.Priority != "high" {
		t.Fatalf("priority: %s", inc.Priority)
	}
}
