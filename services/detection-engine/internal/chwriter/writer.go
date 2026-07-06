// Package chwriter — запись detections в ClickHouse.
package chwriter

import (
	"context"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type Writer struct {
	conn driver.Conn
}

type DetectionRow struct {
	DetectionID string
	EventID     string
	ObservedAt  time.Time
	TenantID    string
	NodeID      string
	RuleID      string
	RuleName    string
	Severity    string
	Engine      string
	Confidence  float32
}

func New(addr, user, password string) (*Writer, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{Database: "era_xdr", Username: user, Password: password},
	})
	if err != nil {
		return nil, err
	}
	if err := conn.Ping(context.Background()); err != nil {
		return nil, err
	}
	return &Writer{conn: conn}, nil
}

func (w *Writer) InsertDetection(ctx context.Context, d DetectionRow) error {
	batch, err := w.conn.PrepareBatch(ctx, `INSERT INTO era_xdr.detections (
		detection_id, event_id, observed_at, tenant_id, node_id,
		rule_id, rule_name, severity, engine, confidence, status
	)`)
	if err != nil {
		return err
	}
	sev := mapSeverity(d.Severity)
	if err := batch.Append(d.DetectionID, d.EventID, d.ObservedAt, d.TenantID, d.NodeID,
		d.RuleID, d.RuleName, sev, d.Engine, d.Confidence, "new"); err != nil {
		return err
	}
	return batch.Send()
}

func mapSeverity(s string) string {
	switch s {
	case "critical", "high", "medium", "low", "info":
		return s
	default:
		return "medium"
	}
}
