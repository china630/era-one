package playbooks

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPlaybooksSimulated(t *testing.T) {
	e := NewFromEnv()
	a1 := e.IsolateHost("node-01")
	if a1.Playbook != "isolate_host" || a1.Status != "completed" {
		t.Fatalf("isolate: %+v", a1)
	}
	a2 := e.BlockIP("10.0.0.5")
	if !e.IsBlocked("10.0.0.5") {
		t.Fatal("ip not blocked")
	}
	_ = a2
	a3 := e.CreateTicket("APT alert", "case-1")
	if a3.Playbook != "create_ticket" {
		t.Fatal(a3)
	}
	if len(e.Actions()) != 3 {
		t.Fatalf("expected 3 actions, got %d", len(e.Actions()))
	}
}

func TestCreateTicketWebhook(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"ticket_id": "EXT-9"})
	}))
	defer srv.Close()
	t.Setenv("ERA_SOAR_TICKET_WEBHOOK", srv.URL)
	t.Setenv("ERA_SOAR_ISOLATE_SCRIPT", "")

	e := NewFromEnv()
	a := e.CreateTicket("incident", "case-99")
	if a.Status != "completed" {
		t.Fatalf("status %s detail %s", a.Status, a.Detail)
	}
	if !strings.Contains(a.Detail, "EXT-9") {
		t.Fatalf("detail %s", a.Detail)
	}
}
