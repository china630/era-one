package main

import (
	"log"
	"net/http"
	"os"

	"era/services/comms/mail-bridge/internal/api"
	"era/services/comms/mail-bridge/internal/audit"
	"era/services/comms/mail-bridge/internal/upstream"
	"era/services/platform/httpserver"
	"era/services/platform/licensegate"
)

func main() {
	gate := licensegate.FromEnv()
	if !gate.Allow(licensegate.ModuleCommsOutlookBridge) {
		if os.Getenv("ERA_BRIDGE_DEV") == "1" {
			gate = licensegate.FromModules([]licensegate.Module{licensegate.ModuleCommsOutlookBridge})
			log.Print("ERA_BRIDGE_DEV=1 — comms-outlook-bridge license bypass")
		} else {
			log.Fatal("license: comms-outlook-bridge not enabled (set ERA_LICENSE_MODULES or ERA_BRIDGE_DEV=1)")
		}
	}
	stub := upstream.StubBackend{}
	router, err := upstream.LoadFromEnv(stub)
	if err != nil {
		log.Fatalf("upstream router: %v", err)
	}
	aud := audit.NewFromEnv()
	srv := api.NewServer(gate, router, aud)
	mux := http.NewServeMux()
	srv.Register(mux)
	addr := env("ERA_BRIDGE_HTTP_ADDR", ":8151")
	log.Printf("era-mail-bridge listening %s", addr)
	if err := httpserver.Listen(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
