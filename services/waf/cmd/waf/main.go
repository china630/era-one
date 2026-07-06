package main

import (
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"era/services/waf/internal/rules"
)

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)
	addr := env("ERA_HTTP_ADDR", ":8093")
	upstream := env("ERA_WAF_UPSTREAM", "http://127.0.0.1:8089")
	engine := rules.NewOWASP()

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, `{"status":"ok"}`)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if m, blocked := engine.Evaluate(r); blocked {
			log.Printf("WAF BLOCK rule=%s cat=%s path=%s", m.RuleID, m.Category, r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"blocked":true,"rule_id":"` + m.RuleID + `"}`))
			return
		}
		proxyTo(w, r, upstream)
	})

	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Printf("waf listening %s (upstream %s)", addr, upstream)
	log.Fatal(srv.ListenAndServe())
}

func proxyTo(w http.ResponseWriter, r *http.Request, upstream string) {
	req := httptest.NewRequest(r.Method, upstream+r.URL.RequestURI(), r.Body)
	req.Header = r.Header.Clone()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write([]byte("proxied"))
}

func writeJSON(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(body))
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
