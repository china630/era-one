package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"era/services/dlp/internal/api"
	"era/services/dlp/internal/session"
	"era/services/platform/envelope"
)

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)
	addr := env("ERA_HTTP_ADDR", ":8095")

	var pub *envelope.Publisher
	if brokers := env("ERA_KAFKA_BROKERS", ""); brokers != "" {
		pub = envelope.New(envelope.MustBrokers(brokers), env("ERA_TENANT_ID", "tenant-dev"), env("ERA_NODE_ID", "dlp-01"), "dlp-uam")
		defer pub.Close()
	}

	srv := api.New(session.NewStore(), pub)
	httpSrv := &http.Server{Addr: addr, Handler: srv.Routes(), ReadHeaderTimeout: 5 * time.Second}
	log.Printf("dlp-uam listening %s", addr)
	log.Fatal(httpSrv.ListenAndServe())
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
