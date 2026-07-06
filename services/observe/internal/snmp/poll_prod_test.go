package snmp

import (
	"os"
	"testing"
)

func TestPollProductionNoSimFallback(t *testing.T) {
	t.Setenv("ERA_PRODUCTION", "1")
	t.Setenv("ERA_OBSERVE_SNMP_SIM", "0")
	m := Poll("203.0.113.99")
	if m.CPUPercent == 42.5 {
		t.Fatal("should not fall back to simulated metrics in production")
	}
	if m.Error == "" && m.CPUPercent != 0 {
		t.Fatal("unexpected success without error in production lab")
	}
}

func TestPollDevSimExplicit(t *testing.T) {
	os.Unsetenv("ERA_PRODUCTION")
	t.Setenv("ERA_OBSERVE_SNMP_SIM", "1")
	m := Poll("any-target")
	if m.CPUPercent != 42.5 {
		t.Fatalf("expected sim metrics, got cpu=%v", m.CPUPercent)
	}
}
