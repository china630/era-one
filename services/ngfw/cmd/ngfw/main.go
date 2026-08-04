package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"era/services/ngfw/internal/api"
	"era/services/ngfw/internal/policy"
	"era/services/platform/envelope"
	"era/services/platform/licensegate"
)

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)
	addr := env("ERA_HTTP_ADDR", ":8094")
	var pub *envelope.Publisher
	if brokers := env("ERA_KAFKA_BROKERS", ""); brokers != "" {
		pub = envelope.New(envelope.MustBrokers(brokers), env("ERA_TENANT_ID", "tenant-dev"), env("ERA_NODE_ID", "ngfw-01"), "ngfw-engine")
		defer pub.Close()
	}
	eng := policy.Default()
	if p := env("ERA_NGFW_POLICY_PATH", ""); p != "" {
		eng.SetPath(p)
		if _, err := os.Stat(p); err == nil {
			if err := eng.Load(p); err != nil {
				log.Printf("policy load: %v (using defaults)", err)
			}
		} else {
			_ = eng.Save(p)
		}
	}
	gate, err := licensegate.GateFromEnv(0)
	if err != nil {
		log.Fatalf("license: %v", err)
	}
	srv := api.New(eng, pub, gate)
	httpSrv := &http.Server{Addr: addr, Handler: srv.Routes(), ReadHeaderTimeout: 5 * time.Second}
	log.Printf("ngfw listening %s (policy decision API, not packet firewall)", addr)
	log.Fatal(httpSrv.ListenAndServe())
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
