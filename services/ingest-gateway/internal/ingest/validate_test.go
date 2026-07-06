package ingest

import (
	"testing"
	"time"

	erav1 "era/contracts/gen/era/v1"
)

func TestValidateAndEnrichOK(t *testing.T) {
	env := &erav1.Envelope{
		SchemaVersion: SupportedSchemaVersion,
		EventId:       []byte("0123456789abcdef"),
		Category:      erav1.EventCategory_EVENT_CATEGORY_PROCESS,
		PiiSanitized:  true,
		Source: &erav1.Source{
			TenantId: "t1",
			NodeId:   "n1",
		},
	}
	res, err := ValidateAndEnrich(env, time.Unix(1_700_000_000, 0))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Topic != "xdr.process" {
		t.Fatalf("topic: got %q", res.Topic)
	}
	if res.Key != "t1|n1" {
		t.Fatalf("key: got %q", res.Key)
	}
	if res.Envelope.GetIngestedAt() == nil {
		t.Fatal("ingested_at not set")
	}
}

func TestValidateInventoryTopic(t *testing.T) {
	env := &erav1.Envelope{
		SchemaVersion: SupportedSchemaVersion,
		EventId:       []byte("0123456789abcdef"),
		Category:      erav1.EventCategory_EVENT_CATEGORY_MODULE,
		PiiSanitized:  true,
		Source:        &erav1.Source{TenantId: "t1", NodeId: "n1"},
		Payload: &erav1.Envelope_Raw{
			Raw: &erav1.RawEvent{SourceType: "era.inventory.host_snapshot"},
		},
	}
	res, err := ValidateAndEnrich(env, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if res.Topic != "xdr.inventory" {
		t.Fatalf("topic: got %q want xdr.inventory", res.Topic)
	}
}

func TestValidateRejectsUnsanitizedPII(t *testing.T) {
	env := &erav1.Envelope{
		SchemaVersion: SupportedSchemaVersion,
		EventId:       []byte("id"),
		PiiSanitized:  false,
		Source:        &erav1.Source{TenantId: "t1", NodeId: "n1"},
	}
	_, err := ValidateAndEnrich(env, time.Now())
	if err == nil {
		t.Fatal("expected error for unsanitized PII")
	}
}
