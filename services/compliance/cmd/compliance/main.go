package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"era/services/compliance/internal/api"
	"era/services/platform/licensegate"
)

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)
	addr := env("ERA_HTTP_ADDR", ":8100")
	gate := licensegate.FromEnv()
	if os.Getenv("ERA_NATIONAL_DEV") == "1" {
		gate = licensegate.FromModules([]licensegate.Module{licensegate.ModuleNational})
	}
	srv := api.New(gate)
	httpSrv := &http.Server{Addr: addr, Handler: srv.Routes(), ReadHeaderTimeout: 5 * time.Second}
	log.Printf("compliance listening %s", addr)
	log.Fatal(httpSrv.ListenAndServe())
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
