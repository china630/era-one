package hybrid

import (
	"strings"
	"time"

	"era/services/control-plane/internal/store"
)

// HealthB — operational aggregates без сырья (ADR-0018 §4, уровень B).
type HealthB struct {
	Level          string    `json:"level"`
	DeploymentID   string    `json:"deployment_id"`
	TenantID       string    `json:"tenant_id,omitempty"`
	At             time.Time `json:"at"`
	AgentCount     int       `json:"agent_count"`
	ActiveAgents   int       `json:"active_agents_24h"`
	OpenCases      int       `json:"open_cases"`
	DetectionRules int       `json:"detection_rules"`
	CoveragePct    float64   `json:"coverage_pct"`
	LeaseStatus    string    `json:"lease_status,omitempty"`
	BundleID       string    `json:"bundle_id,omitempty"`
	PolicyVersion  string    `json:"policy_version"`
	EnginesActive  []string  `json:"engines_active,omitempty"`
}

// BuildHealthB собирает health B из store (агрегаты, без PII/сырья).
func BuildHealthB(st store.Repository, pol store.HybridPolicy, rt store.HybridRuntime) HealthB {
	cutoff := time.Now().UTC().Add(-24 * time.Hour)
	active := 0
	for _, a := range st.ListAssets() {
		if a.LastSeen.After(cutoff) {
			active++
		}
	}
	openCases := 0
	for _, c := range st.ListCases() {
		if c.Status != "closed" && c.Status != "resolved" {
			openCases++
		}
	}
	policy := st.Policy()
	rules := len(policy.Rules)
	engines := make([]string, 0, len(policy.Rules))
	for k := range policy.Rules {
		engines = append(engines, k)
	}
	return HealthB{
		Level:          "B",
		DeploymentID:   pol.DeploymentID,
		TenantID:       pol.TenantID,
		At:             time.Now().UTC(),
		AgentCount:     len(st.ListAssets()),
		ActiveAgents:   active,
		OpenCases:      openCases,
		DetectionRules: rules,
		CoveragePct:    st.AssetCoverage() * 100,
		LeaseStatus:    rt.LeaseStatus,
		BundleID:       rt.LastBundleID,
		PolicyVersion:  policy.Version,
		EnginesActive:  engines,
	}
}

// HealthBForbiddenFields — поля, запрещённые в egress Health B.
func HealthBForbiddenFields(body string) bool {
	lower := strings.ToLower(body)
	for _, forbidden := range []string{
		"cmdline", "command_line", "raw_event", "pii", "lake", "case_body", "note",
		"hostname", "email", "password", "secret",
	} {
		if strings.Contains(lower, forbidden) {
			return true
		}
	}
	return false
}
