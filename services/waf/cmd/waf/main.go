package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"era/services/platform/envelope"
	"era/services/platform/licensegate"
	"era/services/waf/internal/api"
	"era/services/waf/internal/rules"
)

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)
	addr := env("ERA_HTTP_ADDR", ":8093")
	upstream := env("ERA_WAF_UPSTREAM", "http://127.0.0.1:8089")
	engine := rules.NewOWASP()
	if p := env("ERA_WAF_RULES_PATH", ""); p != "" {
		if err := engine.LoadFile(p); err != nil {
			log.Fatalf("load rules: %v", err)
		}
	}
	gate, err := licensegate.GateFromEnv(0)
	if err != nil {
		log.Fatalf("license: %v", err)
	}
	var pub *envelope.Publisher
	if brokers := env("ERA_KAFKA_BROKERS", ""); brokers != "" {
		pub = envelope.New(envelope.MustBrokers(brokers), env("ERA_TENANT_ID", "tenant-dev"), env("ERA_NODE_ID", "waf-01"), "waf")
		defer pub.Close()
	}
	srv, err := api.New(engine, upstream, gate, pub, api.ParseBodyLimit(env("ERA_WAF_BODY_LIMIT", "")))
	if err != nil {
		log.Fatal(err)
	}
	httpSrv := &http.Server{Addr: addr, Handler: srv.Routes(), ReadHeaderTimeout: 5 * time.Second}
	log.Printf("waf listening %s (upstream %s)", addr, upstream)
	log.Fatal(httpSrv.ListenAndServe())
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
