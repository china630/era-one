// Package deception — honeytoken/decoy detection rules (ADR-0006 G-02).
package deception

import "strings"

type Hit struct {
	RuleID  string `json:"rule_id"`
	Title   string `json:"title"`
	Severity string `json:"severity"`
}

// MatchHoneytoken срабатывает на обращение к decoy-ресурсу.
func MatchHoneytoken(payload string) (bool, Hit) {
	low := strings.ToLower(payload)
	if strings.Contains(low, "honeytoken") || strings.Contains(low, "decoy-share") {
		return true, Hit{RuleID: "era-deception-honeytoken", Title: "Honeytoken access", Severity: "high"}
	}
	return false, Hit{}
}
