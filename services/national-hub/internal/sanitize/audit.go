// Package sanitize — аудит экспорта: PII не покидает контур (F4-2).
package sanitize

import (
	"encoding/json"
	"fmt"
	"strings"

	"era/services/national-hub/internal/stix"
	"era/services/platform/envelope"
)

var piiMarkers = []string{
	"@email", "password=", "passport", "alice", "bob@",
	"phone", "ssn", "fin@", "user=", "command_line",
}

// AuditBundle проверяет STIX bundle перед публикацией в нацхаб.
func AuditBundle(raw []byte) error {
	if err := envelope.ValidateNoPII(string(raw)); err != nil {
		return fmt.Errorf("export blocked: %w", err)
	}
	low := strings.ToLower(string(raw))
	for _, m := range piiMarkers {
		if strings.Contains(low, m) {
			return fmt.Errorf("export blocked: PII marker %q", m)
		}
	}
	var b stix.Bundle
	if err := json.Unmarshal(raw, &b); err != nil {
		return fmt.Errorf("invalid stix bundle: %w", err)
	}
	for _, obj := range b.Objects {
		if obj.PatternType != "" && obj.PatternType != "stix" {
			return fmt.Errorf("unsupported pattern_type %q", obj.PatternType)
		}
	}
	return nil
}
