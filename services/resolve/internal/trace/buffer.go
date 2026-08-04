package trace

import (
	"context"
	"strings"
	"sync"

	erav1 "era/contracts/gen/era/v1"
	"era/services/platform/envelope"
	"era/services/resolve/internal/guard"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Record is a Trace entry.
type Record struct {
	Verdict guard.Verdict `json:"verdict"`
}

// Buffer keeps recent DNS decisions and builds Envelopes.
type Buffer struct {
	mu   sync.Mutex
	ring []Record
	max  int
	Pub  *envelope.Publisher
}

func New(max int, pub *envelope.Publisher) *Buffer {
	if max <= 0 {
		max = 256
	}
	return &Buffer{max: max, Pub: pub}
}

func (b *Buffer) Record(v guard.Verdict) *erav1.Envelope {
	answers := []string{string(v.Action)}
	if v.Sinkhole != "" {
		answers = append(answers, v.Sinkhole)
	}
	env := &erav1.Envelope{
		Category:   erav1.EventCategory_EVENT_CATEGORY_DNS,
		Severity:   erav1.Severity_SEVERITY_INFO,
		ObservedAt: timestamppb.Now(),
		Payload: &erav1.Envelope_Dns{
			Dns: &erav1.DnsEvent{Query: v.QName, QueryType: v.QType, Answers: answers},
		},
	}
	if v.Action != "allow" {
		env.Severity = erav1.Severity_SEVERITY_HIGH
	}
	b.mu.Lock()
	b.ring = append(b.ring, Record{Verdict: v})
	if len(b.ring) > b.max {
		b.ring = b.ring[len(b.ring)-b.max:]
	}
	b.mu.Unlock()
	if b.Pub != nil {
		_ = b.Pub.PublishDns(context.Background(), v.QName, v.QType, answers)
	}
	return env
}

func (b *Buffer) Recent(n int) []Record {
	return b.Filter(n, "")
}

// Filter returns up to n recent records; q filters by qname substring (case-insensitive).
func (b *Buffer) Filter(n int, q string) []Record {
	b.mu.Lock()
	defer b.mu.Unlock()
	q = strings.ToLower(strings.TrimSpace(q))
	src := b.ring
	if q != "" {
		filtered := make([]Record, 0, len(src))
		for _, r := range src {
			if strings.Contains(strings.ToLower(r.Verdict.QName), q) {
				filtered = append(filtered, r)
			}
		}
		src = filtered
	}
	if n <= 0 || n > len(src) {
		n = len(src)
	}
	out := make([]Record, n)
	copy(out, src[len(src)-n:])
	return out
}
