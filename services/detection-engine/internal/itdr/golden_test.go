package itdr

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type goldenAuthEvent struct {
	ID         string         `json:"id"`
	Payload    map[string]any `json:"payload"`
	ExpectRule string         `json:"expect_rule"`
}

func TestAuthEventsGolden(t *testing.T) {
	path := filepath.Join("testdata", "auth_events.golden.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var events []goldenAuthEvent
	if err := json.Unmarshal(data, &events); err != nil {
		t.Fatal(err)
	}
	if len(events) < 5 {
		t.Fatalf("events=%d", len(events))
	}
	for _, ev := range events {
		body, _ := json.Marshal(ev.Payload)
		ok, rule := MatchAuth(string(body))
		if !ok || rule.ID != ev.ExpectRule {
			t.Fatalf("%s: ok=%v rule=%s want %s", ev.ID, ok, rule.ID, ev.ExpectRule)
		}
	}
}

func TestASREPRoasting(t *testing.T) {
	payload := `{"event":"AS-REP","kerberos":true,"preauth":"disabled","dont_req_preauth":true}`
	ok, r := MatchAuth(payload)
	if !ok || r.ID != "era-itdr-asrep-roasting" {
		t.Fatalf("ok=%v rule=%s", ok, r.ID)
	}
}

func TestSilverTicket(t *testing.T) {
	payload := `{"event":"kerberos TGS","service":"cifs","spn":"cifs/dc","silver_ticket":true,"rc4":true}`
	ok, r := MatchAuth(payload)
	if !ok || r.ID != "era-itdr-silver-ticket" {
		t.Fatalf("ok=%v rule=%s", ok, r.ID)
	}
}
