package main

import (
	"log"
	"net/http"

	"era/services/comms/mail-connect/internal/api"
	"era/services/comms/mail-connect/internal/audit"
	syncstore "era/services/comms/mail-connect/internal/sync"
	"era/services/platform/httpserver"
)

func main() {
	store := syncstore.NewStore()
	aud := audit.NewRecorder()
	srv := api.NewServer(store, aud)
	mux := http.NewServeMux()
	srv.Register(mux)

	log.Print("era-mail-connect listening :8250")
	if err := httpserver.Listen(":8250", mux); err != nil {
		log.Fatal(err)
	}
}
