// Package publisher — VM findings → Envelope → Kafka (F2-5).
package publisher

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	erav1 "era/contracts/gen/era/v1"
	"github.com/oklog/ulid"
	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/compress"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"era/services/vm/internal/models"
)

type Publisher struct {
	writer *kafka.Writer
	tenant string
	node   string
}

func New(brokers []string, tenant, node string) *Publisher {
	return &Publisher{
		writer: &kafka.Writer{
			Addr:                   kafka.TCP(brokers...),
			Topic:                  "xdr.raw",
			Balancer:               &kafka.Hash{},
			Compression:            compress.Zstd,
			AllowAutoTopicCreation: true,
			RequiredAcks:           kafka.RequireAll,
		},
		tenant: tenant,
		node:   node,
	}
}

func (p *Publisher) Close() error {
	if p == nil || p.writer == nil {
		return nil
	}
	return p.writer.Close()
}

func (p *Publisher) PublishFindings(ctx context.Context, findings []models.Finding) error {
	for _, f := range findings {
		fields, _ := structpb.NewStruct(map[string]any{
			"template_id":        f.TemplateID,
			"target":             f.Target,
			"severity":           f.Severity,
			"vulnerability_name": f.VulnerabilityName,
			"matched_url":        f.MatchedURL,
		})
		id := ulid.MustNew(ulid.Now(), rand.Reader)
		env := &erav1.Envelope{
			SchemaVersion: "1.0.0",
			EventId:       id[:],
			ObservedAt:    timestamppb.New(f.Timestamp.UTC()),
			Source: &erav1.Source{
				TenantId:     p.tenant,
				NodeId:       p.node,
				AgentId:      "vm-engine",
				AgentVersion: "0.1.0",
			},
			Severity:   erav1.Severity_SEVERITY_HIGH,
			Category:   erav1.EventCategory_EVENT_CATEGORY_UNSPECIFIED,
			PiiSanitized: true,
			Payload: &erav1.Envelope_Raw{
				Raw: &erav1.RawEvent{
					SourceType: "vm.finding",
					Fields:     fields,
				},
			},
		}
		data, err := proto.Marshal(env)
		if err != nil {
			return err
		}
		key := p.tenant + "|" + p.node
		if err := p.writer.WriteMessages(ctx, kafka.Message{Key: []byte(key), Value: data}); err != nil {
			return fmt.Errorf("kafka publish finding: %w", err)
		}
	}
	return nil
}

// PublishSmoke — одно тестовое finding без сканирования.
func (p *Publisher) PublishSmoke(ctx context.Context) error {
	return p.PublishFindings(ctx, []models.Finding{{
		TemplateID:        "era-vm-smoke",
		Target:            "127.0.0.1",
		Severity:          "high",
		VulnerabilityName: "smoke-test",
		MatchedURL:        "http://127.0.0.1/smoke",
		Timestamp:         time.Now().UTC(),
	}})
}
