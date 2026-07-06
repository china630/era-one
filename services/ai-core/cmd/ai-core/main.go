package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"era/services/ai-core/internal/api"
	"era/services/ai-core/internal/investigate"
	"era/services/ai-core/internal/llm"
	"era/services/platform/licensegate"
)

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)
	addr := env("ERA_HTTP_ADDR", ":8091")
	chAddr := env("ERA_CH_ADDR", "localhost:9000")

	llmClient := llm.FromEnv()
	if llmClient.Available() {
		log.Printf("ai-core: on-prem LLM available")
	}

	inv, err := investigate.New(chAddr, env("ERA_CH_USER", "era"), env("ERA_CH_PASSWORD", "era_dev_pw"), llmClient)
	if err != nil {
		log.Fatalf("clickhouse: %v", err)
	}
	srv := api.New(inv, licensegate.DevDefault())
	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           srv.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("ai-core слушает %s (POST /api/v1/investigate)", addr)
	log.Fatal(httpSrv.ListenAndServe())
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
