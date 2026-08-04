package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"era/services/platform/licensegate"
	"era/services/soar/internal/api"
	"era/services/soar/internal/playbooks"
)

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)
	addr := env("ERA_HTTP_ADDR", ":8092")
	eng := playbooks.New()
	// Fail-closed when ERA_PRODUCTION / ERA_LICENSE_STRICT without ERA_LICENSE_TOKEN.
	gate, err := licensegate.GateFromEnv(0)
	if err != nil {
		log.Fatalf("license: %v", err)
	}
	srv := api.New(eng, gate)
	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           srv.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("soar слушает %s", addr)
	log.Fatal(httpSrv.ListenAndServe())
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
