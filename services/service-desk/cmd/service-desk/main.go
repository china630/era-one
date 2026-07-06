package main

import (
	"context"
	"log"
	"os"
	"time"

	"era/services/platform/cpclient"
	"era/services/platform/httpserver"
	"era/services/platform/licensegate"
	"era/services/service-desk/internal/api"
	"era/services/service-desk/internal/sla"
	"era/services/service-desk/internal/store"
)

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)
	addr := env("ERA_HTTP_ADDR", ":8122")
	st, err := store.NewFromEnv()
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	if c, ok := st.(store.CloseableRepository); ok {
		defer func() { _ = c.Close() }()
	}
	cp := cpclient.New(env("ERA_CONTROL_PLANE_URL", "http://127.0.0.1:8090"))
	srv := api.New(st, licensegate.DevDefault(), cp)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go runSLAChecker(ctx, st)

	log.Printf("service-desk listening %s (driver=%s)", addr, env("ERA_STORE_DRIVER", "memory"))
	log.Fatal(httpserver.Listen(addr, srv.Routes()))
}

func runSLAChecker(ctx context.Context, st store.Repository) {
	eng := sla.NewEngine(st)
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	eng.CheckBreaches()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if n := eng.CheckBreaches(); len(n) > 0 {
				log.Printf("sla: %d breach(es)", len(n))
			}
		}
	}
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
