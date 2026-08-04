// ERA Identity API — OIDC for webmail (R-3, ADR-0028).
package main

import (
	"log"
	"net/http"
	"os"

	"era/services/platform/httpserver"
	"era/services/platform/internal/oidc"
)

func main() {
	issuer := env("ERA_IDENTITY_ISSUER", "http://127.0.0.1:8160")
	secret := []byte(env("ERA_IDENTITY_JWT_SECRET", "dev-only-change-in-prod"))
	dbURL := env("ERA_COMMS_DATABASE_URL", "")
	srv, err := oidc.NewServer(issuer, secret, dbURL)
	if err != nil {
		log.Fatalf("oidc server: %v", err)
	}

	mux := http.NewServeMux()
	srv.Register(mux)

	addr := env("ERA_IDENTITY_HTTP_ADDR", ":8160")
	log.Printf("identity-api listening %s", addr)
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
