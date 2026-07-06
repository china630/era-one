package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"era/services/ctem/internal/bas"
	"era/services/platform/envelope"
)

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)
	addr := env("ERA_HTTP_ADDR", ":8098")

	var pub *envelope.Publisher
	if brokers := env("ERA_KAFKA_BROKERS", ""); brokers != "" {
		pub = envelope.New(envelope.MustBrokers(brokers), env("ERA_TENANT_ID", "tenant-dev"), env("ERA_NODE_ID", "ctem-01"), "ctem-bas")
		defer pub.Close()
	}
	runner := &bas.Runner{Pub: pub}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/api/v1/bas/lateral", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			SrcIP string `json:"src_ip"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.SrcIP == "" {
			req.SrcIP = "10.0.0.99"
		}
		if err := runner.SimulateLateral(r.Context(), req.SrcIP); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "simulated", "scenario": "T1021-lateral"})
	})

	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Printf("ctem-bas listening %s", addr)
	log.Fatal(srv.ListenAndServe())
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
