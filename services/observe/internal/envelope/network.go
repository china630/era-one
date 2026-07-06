// Package envelope — построение network Envelope для ingest (ADR-0001).
package envelope

import (
	"time"

	erav1 "era/contracts/gen/era/v1"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const SchemaVersion = "1.0.0"

// FromNMSAlert — PRTG/Zabbix/Observe alert → network event.
// Summary в Direction, source в Protocol — для корреляции в detection-engine.
func FromNMSAlert(tenantID, nodeID, source, summary, detail string) *erav1.Envelope {
	id := uuid.New()
	return &erav1.Envelope{
		SchemaVersion: SchemaVersion,
		EventId:       id[:],
		Category:      erav1.EventCategory_EVENT_CATEGORY_NETWORK,
		Severity:      erav1.Severity_SEVERITY_MEDIUM,
		PiiSanitized:  true,
		ObservedAt:    timestamppb.New(time.Now().UTC()),
		Source: &erav1.Source{
			TenantId: tenantID,
			NodeId:   nodeID,
			AgentId:  "era-observe",
		},
		Payload: &erav1.Envelope_Network{
			Network: &erav1.NetworkEvent{
				Protocol:  source,
				Direction: summary,
				SrcIp:     detail,
				BytesSent: 9_500_000,
			},
		},
	}
}
