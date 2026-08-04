package audit

import (
	"context"
	"crypto/rand"
	"os"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/oklog/ulid"
)

type Recorder struct {
	conn driver.Conn
}

func NewFromEnv() *Recorder {
	addr := os.Getenv("ERA_CH_ADDR")
	if addr == "" {
		return &Recorder{}
	}
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{
			Database: "era_comms",
			Username: envOr("ERA_CH_USER", "era"),
			Password: envOr("ERA_CH_PASSWORD", ""),
		},
	})
	if err != nil {
		return &Recorder{}
	}
	if err := conn.Ping(context.Background()); err != nil {
		return &Recorder{}
	}
	return &Recorder{conn: conn}
}

func (r *Recorder) Record(action, mailbox string) {
	if r == nil || r.conn == nil {
		return
	}
	evID := ulid.MustNew(ulid.Now(), rand.Reader).String()
	meta := map[string]string{"service": "mail-bridge"}
	batch, err := r.conn.PrepareBatch(context.Background(), `INSERT INTO era_comms.mail_audit (
		event_id, schema_version, observed_at, tenant_id, mailbox, action, message_id, src_ip, metadata
	)`)
	if err != nil {
		return
	}
	defer batch.Abort()
	_ = batch.Append(evID, "1.0.0", time.Now().UTC(), "t-bridge", mailbox, action, "", "0.0.0.0", meta)
	_ = batch.Send()
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
