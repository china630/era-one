package hybrid

import (
	"regexp"
	"strings"
	"time"

	"era/services/control-plane/internal/store"
)

var piiPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`),
	regexp.MustCompile(`\b\d{1,3}(?:\.\d{1,3}){3}\b`),
	regexp.MustCompile(`(?i)\b(?:cmdline|command_line|password|secret|token)\b`),
}

// HealthA — минимальный health payload (ADR-0018 §4, уровень A).
type HealthA struct {
	Level        string    `json:"level"`
	DeploymentID string    `json:"deployment_id"`
	TenantID     string    `json:"tenant_id,omitempty"`
	At           time.Time `json:"at"`
	AgentCount   int       `json:"agent_count"`
	CaseCount    int       `json:"case_count"`
	Coverage     float64   `json:"coverage"`
	PolicyVer    string    `json:"policy_version"`
	LeaseStatus  string    `json:"lease_status,omitempty"`
	BundleID     string    `json:"bundle_id,omitempty"`
}

// BuildHealthA собирает health A из store (без сырья/PII).
func BuildHealthA(st store.Repository, pol store.HybridPolicy, rt store.HybridRuntime) HealthA {
	h := HealthA{
		Level:        "A",
		DeploymentID: pol.DeploymentID,
		TenantID:     pol.TenantID,
		At:           time.Now().UTC(),
		AgentCount:   len(st.ListAssets()),
		CaseCount:    len(st.ListCases()),
		Coverage:     st.AssetCoverage(),
		PolicyVer:    st.Policy().Version,
		LeaseStatus:  rt.LeaseStatus,
		BundleID:     rt.LastBundleID,
	}
	return h
}

// RedactHealthJSON удаляет PII-подобные фрагменты из сериализованного JSON (defense in depth).
func RedactHealthJSON(raw string) string {
	out := raw
	for _, re := range piiPatterns {
		out = re.ReplaceAllString(out, "[REDACTED]")
	}
	out = strings.ReplaceAll(out, `\u`, `[REDACTED]`)
	return out
}

// ContainsForbiddenEgress проверяет, что payload не содержит запрещённые поля.
func ContainsForbiddenEgress(body string) bool {
	lower := strings.ToLower(body)
	for _, forbidden := range []string{
		"cmdline", "command_line", "raw_event", "pii", "lake", "case_body", "note",
	} {
		if strings.Contains(lower, forbidden) {
			return true
		}
	}
	return false
}
