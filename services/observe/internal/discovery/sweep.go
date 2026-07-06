// Package discovery — ping/ARP sweep (MVP sim; осторожно с сетевыми политиками в проде).
package discovery

import "strings"

type Node struct {
	IP       string   `json:"ip"`
	Hostname string   `json:"hostname,omitempty"`
	MAC      string   `json:"mac,omitempty"`
	Kind     string   `json:"kind"` // switch, printer, iot, unknown
}

// SweepSimulated — demo discovery без raw sockets.
func SweepSimulated(cidr string) []Node {
	if cidr == "" {
		cidr = "10.0.0.0/24"
	}
	base := strings.TrimSuffix(cidr, "/24")
	if !strings.Contains(base, ".") {
		base = "10.0.0"
	}
	prefix := strings.TrimSuffix(base, ".0")
	return []Node{
		{IP: prefix + ".1", Hostname: "switch-core", MAC: "00:11:22:33:44:01", Kind: "switch"},
		{IP: prefix + ".50", Hostname: "printer-hr", MAC: "00:11:22:33:44:50", Kind: "printer"},
		{IP: prefix + ".200", Hostname: "iot-sensor", MAC: "aa:bb:cc:dd:ee:01", Kind: "iot"},
	}
}
