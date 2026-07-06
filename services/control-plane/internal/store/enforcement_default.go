package store

import (
	"encoding/json"

	_ "embed"
)

//go:embed testdata/enforcement_policy.json
var defaultEnforcementPolicyJSON []byte

// DefaultEnforcementPolicy — dev/monitor default (совпадает с era-agent-core testdata).
func DefaultEnforcementPolicy() EnforcementPolicy {
	var p EnforcementPolicy
	if err := json.Unmarshal(defaultEnforcementPolicyJSON, &p); err != nil {
		return EnforcementPolicy{
			Version:  "1.0.0-enforce-dev",
			Mode:     "monitor",
			FailMode: "open",
		}
	}
	return p
}
