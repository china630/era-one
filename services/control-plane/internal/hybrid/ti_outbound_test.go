package hybrid

import "testing"

func TestTIOutboundGolden(t *testing.T) {
	a := BuildTIOutboundAudit("ioc-hash", "evil.example.com/path", "salt-1")
	b := BuildTIOutboundAudit("ioc-hash", "evil.example.com/path", "salt-1")
	if a.PseudonymID != b.PseudonymID || a.Bytes != len("evil.example.com/path") {
		t.Fatalf("a=%+v", a)
	}
	if a.PseudonymID == "evil.example.com/path" {
		t.Fatal("raw IOC leaked")
	}
	if HealthBForbiddenFields(a.PseudonymID) {
		t.Fatal("pseudonym flagged as forbidden")
	}
}
