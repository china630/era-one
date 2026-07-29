package main

import (
	"log"
	"net/http"
	"os"

	"era/services/comms/ai/internal/api"
	"era/services/comms/ai/internal/audit"
	"era/services/comms/ai/internal/llm"
	"era/services/comms/auditch"
	"era/services/platform/httpserver"
	"era/services/platform/licensegate"
)

func main() {
	gate := licensegate.FromEnv()
	if !gate.Allow(licensegate.ModuleCommsAI) {
		if os.Getenv("ERA_COMMS_AI_DEV") == "1" {
			gate = licensegate.FromModules([]licensegate.Module{licensegate.ModuleCommsAI})
			log.Print("ERA_COMMS_AI_DEV=1 — comms-ai license bypass")
		} else {
			log.Fatal("license: comms-ai not enabled (set ERA_LICENSE_MODULES or ERA_COMMS_AI_DEV=1)")
		}
	}

	ch := auditch.NewFromEnv()
	aud := audit.NewRecorder(ch)
	mux := http.NewServeMux()
	api.NewServer(llm.FromEnv(), gate, aud).Register(mux)

	addr := env("ERA_COMMS_AI_ADDR", ":8096")
	log.Printf("era-comms-ai listening %s", addr)
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
