// ERA Office — roadmap stub service (ADR-0024).
//
// Возвращает healthz и заглушку API до PRD/ADR.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"era/services/platform/httpserver"
)

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)
	addr := env("ERA_HTTP_ADDR", ":8142")
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthz)
	mux.HandleFunc("/api/v1/status", status)
	mux.HandleFunc("/", index)
	log.Printf("era-office (docs stub) listening %s [roadmap]", addr)
	log.Fatal(httpserver.Listen(addr, mux))
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]string{"status": "ok", "service": "era-office-docs", "phase": "roadmap"})
}

func status(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]any{
		"product":  "era-office",
		"status":   "roadmap",
		"editions": []string{"office-docs", "office-sheets", "office-slides"},
		"message":  "PRD/ADR pending — donor specs from product team",
	})
}

func index(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!DOCTYPE html><html lang="ru"><head><meta charset="UTF-8">
<title>ERA Office</title></head><body>
<h1>ERA Office</h1>
<p>Roadmap stub. API: <code>/api/v1/status</code></p>
</body></html>`))
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
