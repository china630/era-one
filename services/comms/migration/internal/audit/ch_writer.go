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

// CHWriter persists migration_job events to ClickHouse.
type CHWriter struct {
	conn driver.Conn
}

func NewCHFromEnv() *CHWriter {
	addr := os.Getenv("ERA_CH_ADDR")
	if addr == "" {
		return nil
	}
	w, err := NewCH(addr)
	if err != nil {
		return nil
	}
	return w
}

func NewCH(addr string) (*CHWriter, error) {
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
		return nil, err
	}
	return &CHWriter{conn: conn}, nil
}

func (w *CHWriter) Record(ev Event) error {
	if w == nil || w.conn == nil {
		return nil
	}
	evID := ulid.MustNew(ulid.Now(), rand.Reader).String()
	meta := map[string]string{}
	if ev.Detail != "" {
		meta["detail"] = ev.Detail
	}
	batch, err := w.conn.PrepareBatch(context.Background(), `INSERT INTO era_comms.migration_job (
		event_id, observed_at, job_id, action, source_uid, mailbox, metadata
	)`)
	if err != nil {
		return err
	}
	defer batch.Abort()
	if err := batch.Append(evID, time.Now().UTC(), ev.JobID, ev.Action, ev.SourceUID, ev.Mailbox, meta); err != nil {
		return err
	}
	return batch.Send()
}

func (w *CHWriter) Count(ctx context.Context) (uint64, error) {
	if w == nil || w.conn == nil {
		return 0, nil
	}
	var n uint64
	err := w.conn.QueryRow(ctx, `SELECT count() FROM era_comms.migration_job`).Scan(&n)
	return n, err
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
