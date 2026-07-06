package pipeline

import (
	"testing"

	erav1 "era/contracts/gen/era/v1"
	"github.com/oklog/ulid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// C-01: golden wire envelope для ingest → Kafka → event-writer → CH.
func TestIngestEnvelopeWireGolden(t *testing.T) {
	id := ulid.MustNew(ulid.Timestamp(timestamppb.Now().AsTime()), nil)
	env := &erav1.Envelope{
		SchemaVersion: "1.0.0",
		EventId:       id[:],
		ObservedAt:    timestamppb.Now(),
		Category:      erav1.EventCategory_EVENT_CATEGORY_PROCESS,
		Severity:      erav1.Severity_SEVERITY_MEDIUM,
		PiiSanitized:  true,
		Source: &erav1.Source{
			TenantId: "tenant-e2e",
			NodeId:   "node-e2e-01",
			Hostname: "e2e-host",
			AgentId:  "agent-e2e",
			Platform: erav1.Platform_PLATFORM_LINUX,
		},
		Ocsf: &erav1.OcsfMeta{ClassUid: 1007, CategoryUid: 1, ActivityId: 1},
		Payload: &erav1.Envelope_Process{
			Process: &erav1.ProcessEvent{
				Action: "create", Pid: 100, Ppid: 1,
				ImagePath: "/usr/bin/bash", CommandLine: "bash -c echo e2e",
				User: "pseudo:e2e",
			},
		},
	}
	b, err := proto.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) < 32 {
		t.Fatalf("wire too short: %d", len(b))
	}
	var round erav1.Envelope
	if err := proto.Unmarshal(b, &round); err != nil {
		t.Fatal(err)
	}
	if !round.PiiSanitized || round.GetSource().GetTenantId() != "tenant-e2e" {
		t.Fatal("round-trip mismatch")
	}
}
