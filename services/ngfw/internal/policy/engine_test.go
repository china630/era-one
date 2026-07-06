package policy

import "testing"

func TestDenyExternalSMB(t *testing.T) {
	e := Default()
	d := e.Evaluate("203.0.113.5", "10.0.0.5", "tcp", 445)
	if d.Allowed {
		t.Fatalf("expected deny, got %+v", d)
	}
	if d.RuleID != "deny-external-smb" {
		t.Fatal(d.RuleID)
	}
}

func TestAllowInternal(t *testing.T) {
	e := Default()
	d := e.Evaluate("10.0.0.12", "10.0.0.50", "tcp", 443)
	if !d.Allowed {
		t.Fatalf("expected allow, got %+v", d)
	}
}
