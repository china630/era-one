package erav1_test

import (
	"testing"

	erav1 "era/contracts/gen/era/v1"
	"google.golang.org/protobuf/proto"
)

func TestEnvelopeRoundtrip(t *testing.T) {
	env := &erav1.Envelope{
		SchemaVersion: "1.0.0",
		EventId:       []byte("0123456789abcdef"),
		Category:      erav1.EventCategory_EVENT_CATEGORY_PROCESS,
		Severity:      erav1.Severity_SEVERITY_MEDIUM,
		PiiSanitized:  true,
	}
	encoded, err := proto.Marshal(env)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded erav1.Envelope
	if err := proto.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.GetSchemaVersion() != "1.0.0" {
		t.Fatalf("schema_version: got %q", decoded.GetSchemaVersion())
	}
	if !decoded.GetPiiSanitized() {
		t.Fatal("expected pii_sanitized=true")
	}
}

func TestIngestServiceRegistered(t *testing.T) {
	if erav1.IngestService_ServiceDesc.ServiceName != "era.v1.IngestService" {
		t.Fatalf("unexpected service: %s", erav1.IngestService_ServiceDesc.ServiceName)
	}
}
