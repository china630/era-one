package auditch

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"time"

	erav1 "era/contracts/gen/era/v1"
	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/oklog/ulid"
)

const schemaVersion = "1.0.0"

type Writer struct {
	conn driver.Conn
}

func NewFromEnv() *Writer {
	addr := os.Getenv("ERA_CH_ADDR")
	if addr == "" {
		return &Writer{}
	}
	w, err := New(addr)
	if err != nil {
		return &Writer{}
	}
	return w
}

func New(addr string) (*Writer, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{
			Database: "era_comms",
			Username: envOr("ERA_CH_USER", "era"),
			Password: envOr("ERA_CH_PASSWORD", "era_dev_pw"),
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

func (w *Writer) Insert(ctx context.Context, service, tenantID, action, roomID, userID string) error {
	if w.conn == nil {
		return nil
	}
	evID := ulid.MustNew(ulid.Now(), rand.Reader).String()
	batch, err := w.conn.PrepareBatch(ctx, `INSERT INTO era_comms.chat_vcs_audit (
		event_id, schema_version, observed_at, tenant_id, service, action, room_id, user_id, metadata
	)`)
	if err != nil {
		return err
	}
	defer batch.Abort()
	if err := batch.Append(evID, schemaVersion, time.Now().UTC(), tenantID, service, action, roomID, userID, map[string]string{}); err != nil {
		return err
	}
	return batch.Send()
}

func (w *Writer) RecordChatMessage(ctx context.Context, tenantID, roomID, userID string) error {
	return w.Insert(ctx, "chat", tenantID, erav1.MailAuditAction_MAIL_AUDIT_CHAT_MESSAGE.String(), roomID, userID)
}

func (w *Writer) RecordVCSJoin(ctx context.Context, tenantID, roomID, userID string) error {
	return w.Insert(ctx, "vcs", tenantID, erav1.MailAuditAction_MAIL_AUDIT_VCS_ROOM_JOIN.String(), roomID, userID)
}

func (w *Writer) RecordAIInference(ctx context.Context, tenantID, mailboxID, inferenceType, model string, riskScore int, latencyMs int64, requestID, bodyHash string) error {
	if w.conn == nil {
		return nil
	}
	evID := ulid.MustNew(ulid.Now(), rand.Reader).String()
	batch, err := w.conn.PrepareBatch(ctx, `INSERT INTO era_comms.ai_inference_audit (
		event_id, schema_version, observed_at, tenant_id, mailbox_id, inference_type, model,
		risk_score, latency_ms, request_id, body_hash, metadata
	)`)
	if err != nil {
		return err
	}
	defer batch.Abort()
	meta := map[string]string{
		"action": erav1.MailAuditAction_MAIL_AUDIT_AI_INFERENCE.String(),
	}
	if err := batch.Append(evID, schemaVersion, time.Now().UTC(), tenantID, mailboxID, inferenceType, model,
		uint8(riskScore), uint32(latencyMs), requestID, bodyHash, meta); err != nil {
		return err
	}
	return batch.Send()
}

func (w *Writer) CountAIInference(ctx context.Context, inferenceType string) (uint64, error) {
	if w.conn == nil {
		return 0, fmt.Errorf("noop writer")
	}
	var n uint64
	err := w.conn.QueryRow(ctx, `SELECT count() FROM era_comms.ai_inference_audit WHERE inference_type = ?`, inferenceType).Scan(&n)
	return n, err
}

func (w *Writer) CountByAction(ctx context.Context, action string) (uint64, error) {
	if w.conn == nil {
		return 0, fmt.Errorf("noop writer")
	}
	var n uint64
	err := w.conn.QueryRow(ctx, `SELECT count() FROM era_comms.chat_vcs_audit WHERE action = ?`, action).Scan(&n)
	return n, err
}

func (w *Writer) ApplyDDL(ctx context.Context) error {
	if w.conn == nil {
		return fmt.Errorf("noop writer")
	}
	stmts := []string{
		`CREATE DATABASE IF NOT EXISTS era_comms`,
		`CREATE TABLE IF NOT EXISTS era_comms.chat_vcs_audit (
			event_id String, schema_version LowCardinality(String), observed_at DateTime64(3, 'UTC'),
			tenant_id LowCardinality(String), service LowCardinality(String), action LowCardinality(String),
			room_id String DEFAULT '', user_id String DEFAULT '', metadata Map(String, String)
		) ENGINE = MergeTree() ORDER BY (tenant_id, observed_at, event_id)`,
		`CREATE TABLE IF NOT EXISTS era_comms.ai_inference_audit (
			event_id String, schema_version LowCardinality(String), observed_at DateTime64(3, 'UTC'),
			tenant_id LowCardinality(String), mailbox_id String DEFAULT '',
			inference_type LowCardinality(String), model LowCardinality(String),
			risk_score UInt8 DEFAULT 0, latency_ms UInt32 DEFAULT 0,
			request_id String DEFAULT '', body_hash String DEFAULT '', metadata Map(String, String)
		) ENGINE = MergeTree() ORDER BY (tenant_id, observed_at, event_id)`,
	}
	for _, stmt := range stmts {
		if err := w.conn.Exec(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
