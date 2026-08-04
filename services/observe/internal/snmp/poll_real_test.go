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

func TestPollSimMetricsSource(t *testing.T) {
	t.Setenv("ERA_OBSERVE_SNMP_SIM", "1")
	m := Poll("10.0.0.1")
	if m.MetricsSource != "sim" {
		t.Fatalf("metrics_source: %q", m.MetricsSource)
	}
}

func TestEstimateLoadFallback(t *testing.T) {
	cpu, mem := estimateLoad([]InterfaceStat{{OutOctets: 12345}})
	if cpu < 0 || mem < 0 {
		t.Fatal("negative")
	}
}

func TestPollRealInvalidHostFallsBack(t *testing.T) {
	t.Setenv("ERA_OBSERVE_SNMP_SIM", "")
	_, err := PollReal("127.0.0.1:1")
	if err == nil {
		t.Fatal("expected error for closed snmp port")
	}
}
