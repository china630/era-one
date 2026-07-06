// Package consumer — чтение Kafka topic-per-domain → ClickHouse (S1-5).
package consumer

import (
	"context"
	"log"
	"strings"

	erav1 "era/contracts/gen/era/v1"
	"era/services/event-writer/internal/chwriter"
	ewcustody "era/services/event-writer/internal/custody"
	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
)

var defaultTopics = []string{
	"xdr.process", "xdr.network", "xdr.file", "xdr.registry",
	"xdr.auth", "xdr.dns", "xdr.module", "xdr.raw", "xdr.inventory",
}

// Runner читает все xdr.* топики и пишет в ClickHouse.
type Runner struct {
	readers []*kafka.Reader
	writer  *chwriter.Writer
	groupID string
}

func New(brokers []string, groupID string, writer *chwriter.Writer) *Runner {
	var readers []*kafka.Reader
	for _, topic := range defaultTopics {
		readers = append(readers, kafka.NewReader(kafka.ReaderConfig{
			Brokers:  brokers,
			GroupID:  groupID,
			Topic:    topic,
			MinBytes: 1,
			MaxBytes: 10e6,
		}))
	}
	return &Runner{readers: readers, writer: writer, groupID: groupID}
}

func (r *Runner) Run(ctx context.Context) error {
	log.Printf("event-writer: %d topics, group=%s", len(r.readers), r.groupID)
	errCh := make(chan error, len(r.readers))
	for _, rd := range r.readers {
		go func(reader *kafka.Reader) {
			for {
				msg, err := reader.ReadMessage(ctx)
				if err != nil {
					if ctx.Err() != nil {
						errCh <- nil
						return
					}
					errCh <- err
					return
				}
				var env erav1.Envelope
				if err := proto.Unmarshal(msg.Value, &env); err != nil {
					log.Printf("skip invalid protobuf topic=%s: %v", reader.Config().Topic, err)
					continue
				}
				_ = ewcustody.SealEnvelope(&env)
				if err := r.writer.Insert(ctx, reader.Config().Topic, &env); err != nil {
					log.Printf("clickhouse insert failed: %v", err)
					continue
				}
			}
		}(rd)
	}
	// ждём первую ошибку или отмену контекста
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

func (r *Runner) Close() error {
	var first error
	for _, rd := range r.readers {
		if err := rd.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func ParseBrokers(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
