// Package itdr — Identity Threat Detection & Response (Kerberoasting, DCSync, Golden Ticket).
package itdr

import (
	"encoding/json"
	"strings"
)

// Rule — детекция по auth-событиям (MITRE T1558, T1003.006, T1558.001).
type Rule struct {
	ID        string
	Title     string
	Technique string
	Level     string
}

var pack = []Rule{
	{ID: "era-itdr-kerberoasting", Title: "Kerberoasting TGS-REQ RC4", Technique: "T1558.003", Level: "high"},
	{ID: "era-itdr-dcsync", Title: "DCSync replication abuse", Technique: "T1003.006", Level: "critical"},
	{ID: "era-itdr-golden-ticket", Title: "Golden Ticket anomaly", Technique: "T1558.001", Level: "critical"},
	{ID: "era-itdr-asrep-roasting", Title: "AS-REP Roasting", Technique: "T1558.004", Level: "high"},
	{ID: "era-itdr-silver-ticket", Title: "Silver Ticket", Technique: "T1558.002", Level: "critical"},
}

// MatchAuth проверяет JSON auth payload на ITDR-индикаторы.
func MatchAuth(payload string) (bool, Rule) {
	low := strings.ToLower(payload)
	if low == "" {
		return false, Rule{}
	}
	for _, r := range pack {
		if matchRule(r.ID, low, payload) {
			return true, r
		}
	}
	return false, Rule{}
}

func matchRule(id, low, raw string) bool {
	switch id {
	case "era-itdr-kerberoasting":
		return containsAll(low, "kerberos", "tgs-req") &&
			(containsAny(low, "rc4", "etype\":23", "etype\": 23", "0x17") ||
				containsAny(low, "spn", "service ticket"))
	case "era-itdr-dcsync":
		return containsAny(low, "dcsync", "drsuapi", "getchangesall", "getchanges") ||
			(containsAll(low, "replication", "directory") && containsAny(low, "ds-replication", "1131f6aa"))
	case "era-itdr-golden-ticket":
		var m map[string]any
		if json.Unmarshal([]byte(raw), &m) == nil {
			if lt, ok := m["ticket_lifetime_hours"].(float64); ok && lt > 240 {
				return containsAny(low, "krbtgt", "golden")
			}
		}
		return containsAll(low, "krbtgt", "ticket") &&
			containsAny(low, "lifetime", "renew_max", "forged", "golden")
	case "era-itdr-asrep-roasting":
		return containsAny(low, "as-rep", "as_rep", "asrep") &&
			containsAny(low, "kerberos", "preauth", "pre-auth") &&
			containsAny(low, "disabled", "dont_req_preauth", "0x400000", "etype")
	case "era-itdr-silver-ticket":
		return containsAny(low, "silver", "silver-ticket", "silver_ticket") ||
			(containsAny(low, "kerberos", "tgs") && containsAny(low, "service", "spn") &&
				containsAny(low, "rc4", "etype\":23", "forged", "anomaly"))
	default:
		return false
	}
}

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !strings.Contains(s, p) {
			return false
		}
	}
	return true
}

func containsAny(s string, parts ...string) bool {
	for _, p := range parts {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
}
