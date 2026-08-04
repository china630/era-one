package audit

import (
	"context"
	"os"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// CHWriter — опциональный sink в era_comms.mail_moderation_event (AC-MM-8).
type CHWriter struct {
	conn driver.Conn
}

// NewCHFromEnv подключается при ERA_CH_ADDR; иначе nil.
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
	evID := ev.EventID
	if evID == "" {
		evID = newID()
	}
	obs := ev.Observed
	if obs.IsZero() {
		obs = time.Now().UTC()
	}
	meta := map[string]string{}
	for k, v := range ev.Meta {
		meta[k] = v
	}
	if ev.RuleID != "" {
		meta["rule_id"] = ev.RuleID
	}
	batch, err := w.conn.PrepareBatch(context.Background(), `INSERT INTO era_comms.mail_moderation_event (
		event_id, observed_at, hold_id, action, sender, rule_id, moderator, metadata
	)`)
	if err != nil {
		return err
	}
	defer batch.Abort()
	if err := batch.Append(evID, obs, ev.HoldID, ev.Action, ev.Sender, ev.RuleID, ev.Moderator, meta); err != nil {
		return err
	}
	return batch.Send()
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
