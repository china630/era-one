package discovery

import (
	"os"
	"testing"
)

func TestSweepProductionNoSimFallback(t *testing.T) {
	t.Setenv("ERA_PRODUCTION", "1")
	t.Setenv("ERA_DISCOVERY_ALLOWLIST", "192.0.2.0/30")
	nodes := Sweep("192.0.2.0/30")
	for _, n := range nodes {
		if n.Hostname == "switch-core" || n.Hostname == "printer-hr" {
			t.Fatalf("simulated discovery node in production: %+v", n)
		}
	}
}

func TestSweepDevFallsBackToSim(t *testing.T) {
	os.Unsetenv("ERA_PRODUCTION")
	os.Unsetenv("ERA_OBSERVE_STRICT")
	nodes := Sweep("10.255.255.0/30")
	if len(nodes) == 0 {
		t.Fatal("expected sim fallback nodes in dev")
	}
}
