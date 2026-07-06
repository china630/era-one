package inventory

import (
	"context"
	"log"
	"strings"

	erav1 "era/contracts/gen/era/v1"
	"era/services/control-plane/internal/store"
	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
)

// Consumer читает xdr.inventory и материализует CMDB (ADR-0011).
type Consumer struct {
	Store  store.Repository
	reader *kafka.Reader
}

func NewConsumer(brokers []string, group string, st store.Repository) *Consumer {
	return &Consumer{
		Store: st,
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers: brokers,
			GroupID: group,
			Topic:   "xdr.inventory",
		}),
	}
}

func (c *Consumer) Run(ctx context.Context) error {
	log.Printf("inventory consumer: topic=xdr.inventory")
	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			return err
		}
		var env erav1.Envelope
		if err := proto.Unmarshal(msg.Value, &env); err != nil {
			log.Printf("inventory skip invalid protobuf: %v", err)
			continue
		}
		snap, ok := SnapshotFromEnvelope(&env)
		if !ok {
			continue
		}
		nodeID, rule, audit := ApplySnapshot(c.Store, snap)
		if audit != "" {
			c.Store.RecordAudit("cmdb.merge_conflict", "inventory-consumer", nodeID, audit)
		}
		if rule != "" && nodeID != "" {
			log.Printf("inventory upsert node=%s rule=%s sw=%d", nodeID, rule, len(snap.Software))
		}
	}
}

func (c *Consumer) Close() error {
	if c.reader != nil {
		return c.reader.Close()
	}
	return nil
}

func ParseBrokers(s string) []string {
	parts := strings.Split(s, ",")
	var out []string
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
