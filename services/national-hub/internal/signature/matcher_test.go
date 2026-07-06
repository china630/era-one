package signature

import (
	"encoding/json"
	"testing"
	"time"

	"era/services/national-hub/internal/stix"
)

func TestDetectionDeltaImproves(t *testing.T) {
	bundle := stix.Bundle{
		Type: stix.BundleType, SpecVersion: stix.SpecVersion,
		Objects: []stix.Indicator{{
			Type: stix.IndicatorType, ID: "indicator--evil",
			SpecVersion: stix.SpecVersion, Pattern: "[domain-name:value='evil.az']",
			PatternType: "stix", Confidence: 90,
			Created: time.Now().UTC(), Modified: time.Now().UTC(),
			ValidFrom: time.Now().UTC(), Name: "AZ national IOC",
		}},
	}
	m := FromBundle(&bundle, "national-hub")
	payload := `{"dst_ip":"10.0.0.1","query":"evil.az"}`
	baseline := 0
	withNat := m.Match(payload)
	if withNat == 0 {
		t.Fatal("expected national match")
	}
	delta := DetectionDelta(baseline, withNat)
	if delta < 1.0 {
		t.Fatalf("expected positive delta, got %v", delta)
	}
	_, _ = json.Marshal(bundle)
}
