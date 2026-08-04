// Package audit — запись mail audit events в ClickHouse (PRD AC-C7).
package audit

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"strings"
	"time"

	erav1 "era/contracts/gen/era/v1"
	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/oklog/ulid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const schemaVersion = "1.0.0"

// Writer пишет MailAuditEvent в era_comms.mail_audit.
type Writer struct {
	conn driver.Conn
}

// NewFromEnv подключается к ClickHouse если ERA_CH_ADDR задан; иначе noop.
// When ERA_MAIL_AUDIT_REQUIRE=1, missing/unreachable CH returns an error (G1-5).
func NewFromEnv() *Writer {
	w, err := NewFromEnvStrict()
	if err != nil {
		return NewNoop()
	}
	return w
}

// NewFromEnvStrict fails when audit is required but ClickHouse is unavailable.
func NewFromEnvStrict() (*Writer, error) {
	require := os.Getenv("ERA_MAIL_AUDIT_REQUIRE") == "1"
	addr := strings.TrimSpace(os.Getenv("ERA_CH_ADDR"))
	if addr == "" {
		if require {
			return nil, fmt.Errorf("ERA_CH_ADDR required when ERA_MAIL_AUDIT_REQUIRE=1")
		}
		return NewNoop(), nil
	}
	w, err := New(addr)
	if err != nil {
		if require {
			return nil, fmt.Errorf("clickhouse audit: %w", err)
		}
		return NewNoop(), nil
	}
	return w, nil
}

// New создаёт writer с подключением к ClickHouse.
func New(addr string) (*Writer, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{
			Database: "era_comms",
			Username: envOr("ERA_CH_USER", "era"),
			Password: envOr("ERA_CH_PASSWORD", ""),
		},
	})
	if err != nil {
		return nil, err
	}
	if err := conn.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("clickhouse ping: %w", err)
	}
	return &Writer{conn: conn}, nil
}

// NewNoop возвращает writer без backend (dev без ClickHouse).
func NewNoop() *Writer {
	return &Writer{}
}

// IsConfigured reports whether ClickHouse backend is active (not noop).
func (w *Writer) IsConfigured() bool {
	return w != nil && w.conn != nil
}

// Ping checks ClickHouse connectivity; noop writer returns nil.
func (w *Writer) Ping(ctx context.Context) error {
	if w == nil || w.conn == nil {
		return nil
	}
	return w.conn.Ping(ctx)
}

// Close закрывает соединение.
func (w *Writer) Close() error {
	if w.conn == nil {
		return nil
	}
	return w.conn.Close()
}

// Insert записывает событие аудита.
func (w *Writer) Insert(ctx context.Context, ev *erav1.MailAuditEvent) error {
	if w.conn == nil {
		return nil
	}
	if ev.SchemaVersion == "" {
		ev.SchemaVersion = schemaVersion
	}
	if ev.EventId == "" {
		ev.EventId = ulid.MustNew(ulid.Now(), rand.Reader).String()
	}
	if ev.ObservedAt == nil {
		ev.ObservedAt = timestamppb.Now()
	}
	batch, err := w.conn.PrepareBatch(ctx, `INSERT INTO era_comms.mail_audit (
		event_id, schema_version, observed_at, tenant_id, mailbox, action, message_id, src_ip, metadata
	)`)
	if err != nil {
		return err
	}
	defer batch.Abort()

	action := ev.Action.String()
	srcIP := strings.TrimSpace(ev.SrcIp)
	if srcIP == "" {
		srcIP = "0.0.0.0"
	}
	if err := batch.Append(
		ev.EventId,
		ev.SchemaVersion,
		ev.ObservedAt.AsTime().UTC(),
		ev.TenantId,
		ev.Mailbox,
		action,
		ev.MessageId,
		srcIP,
		ev.Metadata,
	); err != nil {
		return err
	}
	return batch.Send()
}

// CountByEventID возвращает число строк audit с данным event_id (integration / E2E).
func (w *Writer) CountByEventID(ctx context.Context, eventID string) (uint64, error) {
	if w.conn == nil {
		return 0, fmt.Errorf("noop writer")
	}
	var n uint64
	err := w.conn.QueryRow(ctx,
		`SELECT count() FROM era_comms.mail_audit WHERE event_id = ?`, eventID,
	).Scan(&n)
	return n, err
}

// CountSendsByMailbox returns send audit rows for mailbox (integration / E2E).
func (w *Writer) CountSendsByMailbox(ctx context.Context, mailbox string) (uint64, error) {
	if w.conn == nil {
		return 0, fmt.Errorf("noop writer")
	}
	var n uint64
	err := w.conn.QueryRow(ctx,
		`SELECT count() FROM era_comms.mail_audit WHERE mailbox = ? AND action LIKE '%SEND%'`, mailbox,
	).Scan(&n)
	return n, err
}

// CountCalendarCreates returns calendar create audit rows (integration / E2E).
func (w *Writer) CountCalendarCreates(ctx context.Context) (uint64, error) {
	if w.conn == nil {
		return 0, fmt.Errorf("noop writer")
	}
	var n uint64
	err := w.conn.QueryRow(ctx,
		`SELECT count() FROM era_comms.mail_audit WHERE action LIKE '%CALENDAR_CREATE%'`,
	).Scan(&n)
	return n, err
}

// ApplyMailAuditDDL applies comms mail audit schema (integration tests).
func (w *Writer) ApplyMailAuditDDL(ctx context.Context) error {
	if w.conn == nil {
		return fmt.Errorf("noop writer")
	}
	stmts := []string{
		`CREATE DATABASE IF NOT EXISTS era_comms`,
		`CREATE TABLE IF NOT EXISTS era_comms.mail_audit (
			event_id String, schema_version LowCardinality(String), observed_at DateTime64(3, 'UTC'),
			tenant_id LowCardinality(String), mailbox String, action LowCardinality(String),
			message_id String DEFAULT '', src_ip IPv4 DEFAULT toIPv4('0.0.0.0'),
			metadata Map(String, String)
		) ENGINE = MergeTree() ORDER BY (tenant_id, observed_at, event_id)`,
	}
	for _, stmt := range stmts {
		if err := w.conn.Exec(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

// RecordSend — helper для SMTP send audit.
func (w *Writer) RecordSend(ctx context.Context, tenantID, mailbox, messageID, srcIP string) error {
	return w.Insert(ctx, &erav1.MailAuditEvent{
		TenantId:  tenantID,
		Mailbox:   mailbox,
		Action:    erav1.MailAuditAction_MAIL_AUDIT_SEND,
		MessageId: messageID,
		SrcIp:     srcIP,
		ObservedAt: timestamppb.New(time.Now().UTC()),
	})
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
