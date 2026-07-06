package sanitize

import (
	"testing"
)

func TestAuditBundleBlocksPII(t *testing.T) {
	raw := []byte(`{"type":"bundle","objects":[{"pattern":"user=alice@corp.az"}]}`)
	if err := AuditBundle(raw); err == nil {
		t.Fatal("expected PII block")
	}
}

func TestAuditBundleAllowsIOC(t *testing.T) {
	raw := []byte(`{
		"type":"bundle","id":"bundle--1","spec_version":"2.1",
		"objects":[{
			"type":"indicator","id":"indicator--1","spec_version":"2.1",
			"pattern":"[domain-name:value='evil.az']",
			"pattern_type":"stix","confidence":80
		}]
	}`)
	if err := AuditBundle(raw); err != nil {
		t.Fatal(err)
	}
}
