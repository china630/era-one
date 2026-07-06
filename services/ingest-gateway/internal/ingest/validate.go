// Package ingest — валидация и обогащение конвертов (ADR-0001, ADR-0008, ADR-0009).
package ingest

import (
	"fmt"
	"strings"
	"time"

	erav1 "era/contracts/gen/era/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const SupportedSchemaVersion = "1.0.0"

// EnrichResult — конверт после валидации и обогащения.
type EnrichResult struct {
	Envelope *erav1.Envelope
	Topic    string
	Key      string // tenant_id|node_id для Kafka partition key
}

// ValidateAndEnrich проверяет конверт, проставляет ingested_at и возвращает топик.
func ValidateAndEnrich(env *erav1.Envelope, now time.Time) (*EnrichResult, error) {
	if env == nil {
		return nil, fmt.Errorf("envelope is nil")
	}
	if env.GetSchemaVersion() != SupportedSchemaVersion {
		return nil, fmt.Errorf("unsupported schema_version %q", env.GetSchemaVersion())
	}
	if len(env.GetEventId()) == 0 {
		return nil, fmt.Errorf("event_id is required")
	}
	if !env.GetPiiSanitized() {
		return nil, fmt.Errorf("pii_sanitized must be true before ingest")
	}
	src := env.GetSource()
	if src == nil || src.GetTenantId() == "" || src.GetNodeId() == "" {
		return nil, fmt.Errorf("source.tenant_id and source.node_id are required")
	}

	env.IngestedAt = timestamppb.New(now.UTC())

	topic, err := topicForEnvelope(env)
	if err != nil {
		return nil, err
	}

	return &EnrichResult{
		Envelope: env,
		Topic:    topic,
		Key:      src.GetTenantId() + "|" + src.GetNodeId(),
	}, nil
}

func topicForEnvelope(env *erav1.Envelope) (string, error) {
	if raw := env.GetRaw(); raw != nil {
		if strings.HasPrefix(raw.GetSourceType(), "era.inventory") {
			return "xdr.inventory", nil
		}
	}
	return topicForCategory(env.GetCategory())
}

func topicForCategory(cat erav1.EventCategory) (string, error) {
	switch cat {
	case erav1.EventCategory_EVENT_CATEGORY_PROCESS:
		return "xdr.process", nil
	case erav1.EventCategory_EVENT_CATEGORY_NETWORK:
		return "xdr.network", nil
	case erav1.EventCategory_EVENT_CATEGORY_FILE:
		return "xdr.file", nil
	case erav1.EventCategory_EVENT_CATEGORY_REGISTRY:
		return "xdr.registry", nil
	case erav1.EventCategory_EVENT_CATEGORY_AUTH:
		return "xdr.auth", nil
	case erav1.EventCategory_EVENT_CATEGORY_DNS:
		return "xdr.dns", nil
	case erav1.EventCategory_EVENT_CATEGORY_MODULE:
		return "xdr.module", nil
	default:
		return "xdr.raw", nil
	}
}
