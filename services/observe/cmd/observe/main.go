package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"era/services/observe/internal/api"
	"era/services/observe/internal/cmdb"
	"era/services/observe/internal/envelope"
	ingestclient "era/services/observe/internal/ingest"
	"era/services/observe/internal/netflow"
	"era/services/observe/internal/poller"
	"era/services/platform/httpserver"
	"era/services/platform/licensegate"
	erav1 "era/contracts/gen/era/v1"
)

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)
	addr := env("ERA_HTTP_ADDR", ":8132")
	tenant := env("ERA_TENANT_ID", "default")
	ing := ingestclient.New(env("ERA_INGEST_URL", "http://ingest-gateway:8089"), tenant)
	cm := cmdb.New(env("ERA_CONTROL_PLANE_URL", "http://control-plane:8090"))
	gate := licensegate.DevAllEnabled()
	srv := api.New(ing, cm, gate, tenant)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	go poller.Run(ctx, poller.Config{CMDB: cm, Ingest: ing, Tenant: tenant})
	if os.Getenv("ERA_NETFLOW_UDP_DISABLE") != "1" {
		go func() {
			err := netflow.ListenUDP(ctx, func(rec netflow.Record) {
				node := rec.SrcIP
				if node == "" {
					node = "netflow-unknown"
				}
				detail := rec.DstIP + " proto=" + rec.Proto
				ev := envelope.FromNMSAlert(tenant, node, "netflow-udp", "flow", detail)
				if err := ing.PostEvents(context.Background(), []*erav1.Envelope{ev}); err != nil {
					log.Printf("netflow udp ingest: %v", err)
				}
			})
			if err != nil {
				log.Printf("netflow UDP listener: %v (set ERA_NETFLOW_UDP_DISABLE=1 to skip)", err)
			}
		}()
	}

	log.Printf("era-observe listening %s ingest=%s", addr, env("ERA_INGEST_URL", ""))
	log.Fatal(httpserver.Listen(addr, srv.Routes()))
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
