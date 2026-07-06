package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"era/services/deception/internal/honey"
	"era/services/platform/envelope"
)

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)
	addr := env("ERA_HTTP_ADDR", ":8097")
	eng := honey.New()

	var pub *envelope.Publisher
	if brokers := env("ERA_KAFKA_BROKERS", ""); brokers != "" {
		pub = envelope.New(envelope.MustBrokers(brokers), env("ERA_TENANT_ID", "tenant-dev"), env("ERA_NODE_ID", "deception-01"), "deception")
		defer pub.Close()
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/api/v1/deception/touches", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"touches": eng.Touches()})
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if !eng.IsDecoy(r.URL.Path) {
			http.NotFound(w, r)
			return
		}
		touch := eng.Record(r.URL.Path, r.RemoteAddr, r.UserAgent())
		log.Printf("DECEPTION touch path=%s ip=%s", touch.Path, touch.RemoteIP)
		if ok, det := eng.MatchTouchRule(touch); ok {
			log.Printf("DECEPTION detection rule=%s severity=%s", det.RuleID, det.Severity)
			if pub != nil {
				_ = pub.Publish(r.Context(), envelope.RawEvent("deception.touch", map[string]string{
					"rule_id": det.RuleID, "severity": det.Severity, "path": touch.Path,
				}))
			}
		} else if pub != nil {
			_ = pub.Publish(r.Context(), envelope.RawEvent("deception.touch", nil))
		}
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"honeypot":true}`))
	})

	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Printf("deception listening %s", addr)
	log.Fatal(srv.ListenAndServe())
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
