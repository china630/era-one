package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"era/services/observe/internal/api"
	"era/services/observe/internal/cmdb"
	ingestclient "era/services/observe/internal/ingest"
	"era/services/observe/internal/poller"
	"era/services/platform/httpserver"
	"era/services/platform/licensegate"
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

	log.Printf("era-observe listening %s ingest=%s", addr, env("ERA_INGEST_URL", ""))
	log.Fatal(httpserver.Listen(addr, srv.Routes()))
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
