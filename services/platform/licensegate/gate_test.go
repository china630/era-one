package licensegate

import "testing"

func TestDevDefaultFederatedOff(t *testing.T) {
	g := DevDefault()
	if g.Allow(ModuleFederated) {
		t.Fatal("federated must be off by default")
	}
	if g.Allow(ModuleNational) {
		t.Fatal("national must be off by default")
	}
	if !g.Allow(ModuleControlAI) {
		t.Fatal("ai should be on in dev default")
	}
}

func TestFromModules(t *testing.T) {
	g := FromModules([]Module{ModuleControlAI})
	if !g.Allow(ModuleControlAI) {
		t.Fatal("expected ai enabled")
	}
	if g.Allow(ModuleResponse) {
		t.Fatal("expected response disabled")
	}
}

func TestDevAllEnabled(t *testing.T) {
	g := DevAllEnabled()
	for _, m := range KnownModules {
		if !g.Allow(m) {
			t.Fatalf("expected %s enabled", m)
		}
	}
}
