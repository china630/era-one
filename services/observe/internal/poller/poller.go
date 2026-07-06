// Package poller — фоновый SNMP poller по targets из CMDB.
package poller

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"era/services/observe/internal/cmdb"
	"era/services/observe/internal/snmp"
	ingestclient "era/services/observe/internal/ingest"
	"era/services/observe/internal/envelope"
	erav1 "era/contracts/gen/era/v1"
)

// Config — параметры фонового poller.
type Config struct {
	CMDB     *cmdb.Client
	Ingest   *ingestclient.Client
	Tenant   string
	Interval time.Duration
	Targets  []string
}

// Run запускает цикл poll до отмены ctx.
func Run(ctx context.Context, cfg Config) {
	if os.Getenv("ERA_OBSERVE_SNMP_SIM") == "" && os.Getenv("ERA_OBSERVE_POLLER") != "1" {
		return
	}
	iv := cfg.Interval
	if iv <= 0 {
		iv = intervalFromEnv()
	}
	ticker := time.NewTicker(iv)
	defer ticker.Stop()
	pollOnce(cfg)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pollOnce(cfg)
		}
	}
}

func intervalFromEnv() time.Duration {
	if s := os.Getenv("ERA_OBSERVE_POLL_INTERVAL_SEC"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			return time.Duration(n) * time.Second
		}
	}
	return 60 * time.Second
}

func pollOnce(cfg Config) {
	targets := cfg.Targets
	if len(targets) == 0 && cfg.CMDB != nil {
		assets, err := cfg.CMDB.ListNetwork(context.Background())
		if err == nil {
			for _, a := range assets {
				for _, ip := range a.IPAddrs {
					if ip != "" {
						targets = append(targets, ip)
					}
				}
			}
		}
	}
	if len(targets) == 0 {
		targets = []string{"10.0.0.1"}
	}
	var events []*erav1.Envelope
	for _, target := range targets {
		m := snmp.Poll(target)
		if ok, msg := snmp.HighEgressAlert(m); ok {
			node := "net-" + strings.ReplaceAll(target, ".", "-")
			events = append(events, envelope.FromNMSAlert(cfg.Tenant, node, "observe_snmp", msg, target))
		}
	}
	if len(events) > 0 && cfg.Ingest != nil {
		if err := cfg.Ingest.PostEvents(context.Background(), events); err != nil {
			log.Printf("observe poller ingest: %v", err)
		}
	}
}
