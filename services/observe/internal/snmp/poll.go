// Package snmp — SNMP poll stub (MVP sim; prod → snmp_exporter pattern).
package snmp

import "time"

type InterfaceStat struct {
	IfIndex int    `json:"if_index"`
	IfName  string `json:"if_name"`
	InOctets uint64 `json:"in_octets"`
	OutOctets uint64 `json:"out_octets"`
	Status  string `json:"status"`
}

type HostMetrics struct {
	Target     string           `json:"target"`
	PolledAt   time.Time        `json:"polled_at"`
	CPUPercent float64          `json:"cpu_percent"`
	MemPercent float64          `json:"mem_percent"`
	Interfaces []InterfaceStat  `json:"interfaces"`
	Error      string           `json:"error,omitempty"`
}

// PollSimulated — dev/demo без реального SNMP (air-gap lab).
func PollSimulated(target string) HostMetrics {
	return HostMetrics{
		Target: target, PolledAt: time.Now().UTC(),
		CPUPercent: 42.5, MemPercent: 61.0,
		Interfaces: []InterfaceStat{
			{IfIndex: 1, IfName: "Gi0/1", InOctets: 1_000_000, OutOctets: 9_500_000, Status: "up"},
			{IfIndex: 2, IfName: "Gi0/2", InOctets: 500_000, OutOctets: 200_000, Status: "up"},
		},
	}
}

func HighEgressAlert(m HostMetrics) (bool, string) {
	for _, iface := range m.Interfaces {
		if iface.OutOctets > 5_000_000 {
			return true, "observe_snmp high egress on " + iface.IfName
		}
	}
	return false, ""
}
