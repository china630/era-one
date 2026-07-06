package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"era/services/federated/internal/api"
	"era/services/federated/internal/hub"
	"era/services/platform/licensegate"
)

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)
	addr := env("ERA_HTTP_ADDR", ":8096")
	gate := licensegate.FromEnv()
	if os.Getenv("ERA_FEDERATED_DEV") == "1" {
		gate = licensegate.FromModules([]licensegate.Module{licensegate.ModuleFederated})
	}
	var h api.HubAPI = hub.New(2.0)
	if storePath := env("ERA_FEDERATED_STORE", ""); storePath != "" {
		store, err := hub.OpenStore(storePath)
		if err != nil {
			log.Fatalf("federated store: %v", err)
		}
		defer store.Close()
		h = hub.NewPersistent(2.0, store)
		log.Printf("federated hub: sqlite %s", storePath)
	}
	srv := api.New(h, gate)
	httpSrv := &http.Server{Addr: addr, Handler: srv.Routes(), ReadHeaderTimeout: 5 * time.Second}
	log.Printf("federated-hub listening %s (federated licensed=%v)", addr, gate.Allow(licensegate.ModuleFederated))
	log.Fatal(httpSrv.ListenAndServe())
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
