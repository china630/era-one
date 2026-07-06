package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"era/services/national-hub/internal/hub"
	"era/services/national-hub/internal/taxii"
	"era/services/platform/licensegate"
)

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)
	addr := env("ERA_HTTP_ADDR", ":8099")
	gate := licensegate.FromEnv()
	if os.Getenv("ERA_NATIONAL_DEV") == "1" {
		gate = licensegate.FromModules([]licensegate.Module{licensegate.ModuleNational})
	}
	st, cleanup, err := hub.NewFromEnv("")
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	defer cleanup()
	srv := taxii.New(st, gate)
	httpSrv := &http.Server{
		Addr: addr, Handler: srv.RoutesWithObjects(), ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("national-hub TAXII listening %s (licensed=%v, persistent=%v)", addr,
		gate.Allow(licensegate.ModuleNational), os.Getenv("ERA_STORE_PATH") != "")
	log.Fatal(httpSrv.ListenAndServe())
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
