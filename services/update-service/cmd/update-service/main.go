package main

import (
	"log"
	"os"

	"era/services/update-service/internal/api"
	"era/services/update-service/internal/bundle"
)

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)
	addr := env("ERA_HTTP_ADDR", ":8110")
	priv, pub, err := bundle.LoadSigningKey()
	if err != nil {
		log.Fatalf("signing key: %v", err)
	}
	srv, err := api.New(priv, pub)
	if err != nil {
		log.Fatalf("bundle: %v", err)
	}
	log.Printf("update-service listening %s (pub fingerprint %x…)", addr, pub[:4])
	log.Fatal(api.ListenAddr(addr, srv.Routes()))
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
