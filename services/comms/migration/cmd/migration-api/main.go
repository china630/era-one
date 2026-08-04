package main

import (
	"log"
	"net/http"

	"era/services/comms/migration/internal/api"
	"era/services/comms/migration/internal/audit"
	"era/services/comms/migration/internal/jobs"
	"era/services/platform/httpserver"
)

func main() {
	mem := audit.NewRecorder()
	ch := audit.NewCHFromEnv()
	var rec audit.Recorder = mem
	if ch != nil {
		rec = &audit.Composite{Mem: mem, CH: ch}
		log.Print("migration: ClickHouse audit enabled")
	}
	store := jobs.NewStore()
	repo, err := jobs.OpenRepositoryFromEnv(store)
	if err != nil {
		log.Fatalf("jobs store: %v", err)
	}
	srv := api.NewServer(repo, rec)
	mux := http.NewServeMux()
	srv.Register(mux)
	log.Print("comms-migration listening :8350")
	if err := httpserver.Listen(":8350", mux); err != nil {
		log.Fatal(err)
	}
}
