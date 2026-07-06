// Package server — HTTP-маршруты ingest-gateway (health, REST fallback, UI API).
package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	erav1 "era/contracts/gen/era/v1"
	"era/services/ingest-gateway/internal/grpcserver"
	"era/services/ingest-gateway/internal/kafka"
	"era/services/platform/metrics"
)

// Config — зависимости HTTP-слоя.
type Config struct {
	Producer   *kafka.Producer
	GRPC       *grpcserver.Server
	EventsAPI  http.Handler // опционально: прокси к ClickHouse (S1-10)
}

// Routes собирает HTTP-маршрутизатор сервиса.
func Routes(cfg Config) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", handleHealth)
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		handleReady(w, r, cfg.Producer)
	})
	mux.Handle("/metrics", metrics.Handler())
	mux.HandleFunc("/v1/ingest", func(w http.ResponseWriter, r *http.Request) {
		handleIngest(w, r, cfg.GRPC)
	})
	if cfg.EventsAPI != nil {
		mux.Handle("/api/events", cfg.EventsAPI)
	}
	mux.Handle("/ui/", http.StripPrefix("/ui/", http.FileServer(http.Dir("ui/events"))))
	return WithRBAC(RBACStrictFromEnv(), mux)
}

type statusResponse struct {
	Status string `json:"status"`
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, statusResponse{Status: "ok"})
}

func handleReady(w http.ResponseWriter, _ *http.Request, producer *kafka.Producer) {
	if producer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := producer.Ping(ctx); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, statusResponse{Status: "kafka unavailable"})
			return
		}
	}
	writeJSON(w, http.StatusOK, statusResponse{Status: "ready"})
}

// ingestRequest — REST-фолбэк: принимает JSON EventBatch-подобный объект.
type ingestRequest struct {
	BatchID  string            `json:"batch_id"`
	AgentID  string            `json:"agent_id"`
	TenantID string            `json:"tenant_id"`
	Events   []*erav1.Envelope `json:"events"`
}

type ingestResponse struct {
	BatchID  string `json:"batch_id"`
	Status   string `json:"status"`
	Accepted int    `json:"accepted"`
	Message  string `json:"message,omitempty"`
}

func handleIngest(w http.ResponseWriter, r *http.Request, grpc *grpcserver.Server) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req ingestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ingestResponse{Status: "REJECTED", Message: "invalid body"})
		return
	}

	batch := &erav1.EventBatch{
		BatchId:  []byte(req.BatchID),
		AgentId:  req.AgentID,
		TenantId: req.TenantID,
		Events:   req.Events,
	}
	ack, err := grpc.PushEvents(r.Context(), batch)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ingestResponse{Status: "RETRY", Message: err.Error()})
		return
	}

	status := "ACCEPTED"
	switch ack.GetStatus() {
	case erav1.Status_STATUS_REJECTED:
		status = "REJECTED"
	case erav1.Status_STATUS_RETRY, erav1.Status_STATUS_THROTTLE:
		status = "RETRY"
	case erav1.Status_STATUS_PARTIAL:
		status = "PARTIAL"
	}

	log.Printf("REST ingest batch=%s status=%s", req.BatchID, status)
	writeJSON(w, http.StatusAccepted, ingestResponse{
		BatchID:  req.BatchID,
		Status:   status,
		Accepted: len(req.Events) - len(ack.GetRejectedEventIds()),
		Message:  ack.GetMessage(),
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
