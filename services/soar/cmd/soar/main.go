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
	srv := api.New(eng, licensegate.DevDefault())
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
