package store

import "testing"

func TestDefaultEnforcementPolicyMatchesGolden(t *testing.T) {
	p := DefaultEnforcementPolicy()
	if p.Version != "1.0.0-enforce-dev" {
		t.Fatalf("version: %s", p.Version)
	}
	if p.Mode != "monitor" {
		t.Fatalf("mode: %s", p.Mode)
	}
	if len(p.AppRules) == 0 || len(p.DeviceRules) == 0 {
		t.Fatal("expected default rules")
	}
}
