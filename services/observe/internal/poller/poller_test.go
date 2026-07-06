package poller

import (
	"context"
	"testing"
	"time"

	ingestclient "era/services/observe/internal/ingest"
)

func TestPollOnceSim(t *testing.T) {
	t.Setenv("ERA_OBSERVE_SNMP_SIM", "1")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go Run(ctx, Config{
		Ingest: ingestclient.New("", "t1"),
		Tenant: "t1",
		Targets: []string{"10.0.0.1"},
		Interval: 100 * time.Millisecond,
	})
	time.Sleep(150 * time.Millisecond)
	cancel()
}
