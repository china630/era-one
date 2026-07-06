package snmp

import (
	"testing"
)

func TestPollUsesSimWhenEnvSet(t *testing.T) {
	t.Setenv("ERA_OBSERVE_SNMP_SIM", "1")
	m := Poll("10.0.0.1")
	if m.CPUPercent != 42.5 {
		t.Fatalf("expected sim metrics, got %+v", m)
	}
}

func TestPollRealInvalidHostFallsBack(t *testing.T) {
	t.Setenv("ERA_OBSERVE_SNMP_SIM", "")
	_, err := PollReal("127.0.0.1:1")
	if err == nil {
		t.Fatal("expected error for closed snmp port")
	}
}
