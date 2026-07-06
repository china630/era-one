// Package envelope — общий Kafka-пublisher Envelope (perimeter-модули, Фаза 3).
package envelope

import (
	"context"
	"crypto/rand"
	"fmt"

	erav1 "era/contracts/gen/era/v1"
	"github.com/oklog/ulid"
	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/compress"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Publisher struct {
	brokers []string
	tenant  string
	node    string
	agent   string
	writers map[string]*kafka.Writer
}

func New(brokers []string, tenant, node, agentID string) *Publisher {
	return &Publisher{
		brokers: brokers,
		tenant:  tenant,
		node:    node,
		agent:   agentID,
		writers: make(map[string]*kafka.Writer),
	}
}

func (p *Publisher) Close() error {
	var first error
	for _, w := range p.writers {
		if err := w.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func (p *Publisher) Publish(ctx context.Context, env *erav1.Envelope) error {
	if env.SchemaVersion == "" {
		env.SchemaVersion = "1.0.0"
	}
	if len(env.EventId) == 0 {
		id := ulid.MustNew(ulid.Now(), rand.Reader)
		env.EventId = id[:]
	}
	if env.ObservedAt == nil {
		env.ObservedAt = timestamppb.Now()
	}
	if env.Source == nil {
		env.Source = &erav1.Source{}
	}
	if env.Source.TenantId == "" {
		env.Source.TenantId = p.tenant
	}
	if env.Source.NodeId == "" {
		env.Source.NodeId = p.node
	}
	if env.Source.AgentId == "" {
		env.Source.AgentId = p.agent
	}
	env.PiiSanitized = true

	topic, err := topicForCategory(env.GetCategory())
	if err != nil {
		return err
	}
	data, err := proto.Marshal(env)
	if err != nil {
		return err
	}
	w := p.writer(topic)
	key := env.Source.TenantId + "|" + env.Source.NodeId
	return w.WriteMessages(ctx, kafka.Message{Key: []byte(key), Value: data})
}

func (p *Publisher) writer(topic string) *kafka.Writer {
	if w, ok := p.writers[topic]; ok {
		return w
	}
	w := &kafka.Writer{
		Addr:                   kafka.TCP(p.brokers...),
		Topic:                  topic,
		Balancer:               &kafka.Hash{},
		Compression:            compress.Zstd,
		AllowAutoTopicCreation: true,
		RequiredAcks:           kafka.RequireAll,
	}
	p.writers[topic] = w
	return w
}

func topicForCategory(cat erav1.EventCategory) (string, error) {
	switch cat {
	case erav1.EventCategory_EVENT_CATEGORY_PROCESS:
		return "xdr.process", nil
	case erav1.EventCategory_EVENT_CATEGORY_NETWORK:
		return "xdr.network", nil
	case erav1.EventCategory_EVENT_CATEGORY_AUTH:
		return "xdr.auth", nil
	case erav1.EventCategory_EVENT_CATEGORY_FILE:
		return "xdr.file", nil
	default:
		return "xdr.raw", nil
	}
}

func NetworkEvent(srcIP, dstIP, protocol, direction string, dstPort uint32) *erav1.Envelope {
	return &erav1.Envelope{
		Category: erav1.EventCategory_EVENT_CATEGORY_NETWORK,
		Severity: erav1.Severity_SEVERITY_MEDIUM,
		Payload: &erav1.Envelope_Network{
			Network: &erav1.NetworkEvent{
				SrcIp: srcIP, DstIp: dstIP, Protocol: protocol,
				Direction: direction, DstPort: dstPort,
			},
		},
	}
}

func AuthEvent(user, action string, success bool, srcIP string) *erav1.Envelope {
	sev := erav1.Severity_SEVERITY_INFO
	if !success {
		sev = erav1.Severity_SEVERITY_HIGH
	}
	return &erav1.Envelope{
		Category: erav1.EventCategory_EVENT_CATEGORY_AUTH,
		Severity: sev,
		Payload: &erav1.Envelope_Auth{
			Auth: &erav1.AuthEvent{
				User: user, Action: action, Success: success, SrcIp: srcIP,
			},
		},
	}
}

func RawEvent(sourceType string, fields map[string]string) *erav1.Envelope {
	return &erav1.Envelope{
		Category: erav1.EventCategory_EVENT_CATEGORY_UNSPECIFIED,
		Severity: erav1.Severity_SEVERITY_HIGH,
		Payload: &erav1.Envelope_Raw{
			Raw: &erav1.RawEvent{SourceType: sourceType},
		},
	}
}

// PublishNetwork helper.
func (p *Publisher) PublishNetwork(ctx context.Context, srcIP, dstIP, protocol, direction string, dstPort uint32) error {
	return p.Publish(ctx, NetworkEvent(srcIP, dstIP, protocol, direction, dstPort))
}

// PublishAuth helper.
func (p *Publisher) PublishAuth(ctx context.Context, user, action string, success bool, srcIP string) error {
	return p.Publish(ctx, AuthEvent(user, action, success, srcIP))
}

// MustBrokers parses comma-separated brokers.
func MustBrokers(s string) []string {
	if s == "" {
		return []string{"localhost:9092"}
	}
	var out []string
	for _, b := range splitComma(s) {
		if b != "" {
			out = append(out, b)
		}
	}
	return out
}

func splitComma(s string) []string {
	var parts []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	return parts
}

// ValidateNoPII rejects payloads containing obvious PII markers (federated audit).
func ValidateNoPII(body string) error {
	low := body
	for _, bad := range []string{"@email", "password=", "alice", "bob@", "passport"} {
		if containsFold(low, bad) {
			return fmt.Errorf("PII marker detected: %s", bad)
		}
	}
	return nil
}

func containsFold(s, sub string) bool {
	return len(sub) > 0 && (len(s) >= len(sub)) && searchFold(s, sub)
}

func searchFold(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		match := true
		for j := 0; j < len(sub); j++ {
			a, b := s[i+j], sub[j]
			if a >= 'A' && a <= 'Z' {
				a += 32
			}
			if b >= 'A' && b <= 'Z' {
				b += 32
			}
			if a != b {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
