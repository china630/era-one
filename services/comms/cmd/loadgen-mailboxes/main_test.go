package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestF_C33_LoadgenScaleProof(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/api/v1/ai/summary", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-ERA-Tenant") == "" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"summary": "ok", "model": "stub", "latency_ms": 1})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	report, err := RunScaleProof(ctx, Config{
		Addr:      srv.URL,
		Mailboxes: 200,
		Workers:   8,
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Errors > 0 {
		t.Fatalf("errors=%d", report.Errors)
	}
	if report.SummaryOK == 0 {
		t.Fatal("no successful summaries")
	}
	if report.Throughput <= 0 {
		t.Fatalf("throughput=%f", report.Throughput)
	}
}
