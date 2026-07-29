package main

import (
	"log"
	"net/http"

	"era/services/comms/vcs/internal/adapter"
	"era/services/comms/vcs/internal/api"
	"era/services/comms/vcs/internal/audit"
	"era/services/comms/vcs/internal/store"
	"era/services/platform/httpserver"
)

func main() {
	mux := http.NewServeMux()
	api.NewServer(store.New(), adapter.Stub{}, audit.NewRecorder()).Register(mux)
	log.Print("era-conference listening :8270")
	if err := httpserver.Listen(":8270", mux); err != nil {
		log.Fatal(err)
	}
}
