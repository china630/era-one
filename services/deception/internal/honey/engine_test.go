package honey

import "testing"

func TestDecoyPaths(t *testing.T) {
	e := New()
	if !e.IsDecoy("/decoy/admin") {
		t.Fatal("expected decoy")
	}
	if e.IsDecoy("/healthz") {
		t.Fatal("unexpected decoy")
	}
	touch := e.Record("/decoy/admin", "203.0.113.1", "scanner")
	if touch.ID == "" {
		t.Fatal("empty id")
	}
}

func TestHoneytokenTouchDetection(t *testing.T) {
	e := New()
	tokens := e.Honeytokens()
	if len(tokens) < 2 {
		t.Fatalf("honeytokens=%d", len(tokens))
	}
	touch := e.Record("/decoy/creds/.env", "198.51.100.7", "curl/8.0")
	ok, det := e.MatchTouchRule(touch)
	if !ok || det.RuleID != "era-deception-honeytoken-touch" {
		t.Fatalf("ok=%v det=%+v", ok, det)
	}
	if det.Severity != "critical" {
		t.Fatalf("severity=%s", det.Severity)
	}
}

func TestDecoyTouchRule(t *testing.T) {
	e := New()
	touch := e.Record("/decoy/backup.sql", "10.0.0.5", "nmap")
	ok, det := e.MatchTouchRule(touch)
	if !ok || det.RuleID != "era-deception-decoy-touch" {
		t.Fatalf("ok=%v det=%+v", ok, det)
	}
}
