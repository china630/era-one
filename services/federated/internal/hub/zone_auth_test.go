package hub

import "testing"

func TestValidateZoneToken(t *testing.T) {
	t.Setenv("ERA_FEDERATED_ZONE_KEY", "zone-secret-xyz")
	if ValidateZoneToken("zone-secret-xyz") != true {
		t.Fatal("expected valid token")
	}
	if ValidateZoneToken("wrong") != false {
		t.Fatal("expected invalid token")
	}
	if ValidateZoneToken("") != false {
		t.Fatal("expected reject empty")
	}
}

func TestSubmissionAuditLog(t *testing.T) {
	h := New(1.0)
	_ = h.Submit(GradientSubmission{ZoneID: "z1", Vector: []float64{0.1, 0.2}, SampleCount: 10})
	if h.AuditEntries()[0].ZoneID != "z1" {
		t.Fatalf("audit zone=%s", h.AuditEntries()[0].ZoneID)
	}
}
