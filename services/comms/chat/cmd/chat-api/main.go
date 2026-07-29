package main

import (
	"log"
	"net/http"

	"era/services/comms/chat/internal/api"
	"era/services/comms/chat/internal/audit"
	"era/services/comms/chat/internal/store"
	"era/services/platform/httpserver"
)

func main() {
	mux := http.NewServeMux()
	api.NewServer(store.New(), audit.NewRecorder()).Register(mux)
	log.Print("era-chat listening :8260")
	if err := httpserver.Listen(":8260", mux); err != nil {
		log.Fatal(err)
	}
}
