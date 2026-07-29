package api

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"era/services/comms/ai/internal/audit"
	"era/services/comms/ai/internal/llm"
	"era/services/comms/ai/internal/phishing"
	"era/services/comms/ai/internal/summary"
	"era/services/platform/licensegate"

	"github.com/oklog/ulid"
)

const summarySLAMs = 5000

type Server struct {
	LLM      llm.Client
	Gate     *licensegate.Gate
	Auditor  *audit.Recorder
}

func NewServer(client llm.Client, gate *licensegate.Gate, aud *audit.Recorder) *Server {
	if client == nil {
		client = llm.Heuristic{}
	}
	return &Server{LLM: client, Gate: gate, Auditor: aud}
}

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("/healthz", s.healthz)
	mux.HandleFunc("/api/v1/ai/summary", s.withRBAC(s.handleSummary))
	mux.HandleFunc("/api/v1/ai/phishing", s.withRBAC(s.handlePhishing))
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "era-comms-ai"})
}

func (s *Server) withRBAC(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !authorize(r) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if s.Gate != nil && !s.Gate.Allow(licensegate.ModuleCommsAI) {
			http.Error(w, "module comms-ai not licensed", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

func authorize(r *http.Request) bool {
	tenant := r.Header.Get("X-ERA-Tenant")
	role := r.Header.Get("X-ERA-Role")
	return tenant != "" && strings.Contains(role, "mail.user")
}

type summaryRequest struct {
	TenantID  string            `json:"tenant_id"`
	MailboxID string            `json:"mailbox_id"`
	Thread    []summary.Message `json:"thread"`
}

func (s *Server) handleSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req summaryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	res, err := summary.Summarize(ctx, s.LLM, req.Thread)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	requestID := ulid.MustNew(ulid.Now(), rand.Reader).String()
	body := concatThread(req.Thread)
	if s.Auditor != nil {
		s.Auditor.Record(ctx, audit.Event{
			TenantID:      firstNonEmpty(req.TenantID, r.Header.Get("X-ERA-Tenant")),
			MailboxID:     req.MailboxID,
			InferenceType: "summary",
			Model:         res.Model,
			LatencyMs:     res.LatencyMs,
			RequestID:     requestID,
			BodyHash:      audit.BodyHash(body),
		})
	}
	writeJSON(w, http.StatusOK, res)
}

type phishingRequest struct {
	TenantID  string           `json:"tenant_id"`
	MailboxID string           `json:"mailbox_id"`
	Message   phishing.Message `json:"message"`
}

func (s *Server) handlePhishing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req phishingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	start := time.Now()
	res := phishing.Classify(req.Message)
	latencyMs := time.Since(start).Milliseconds()

	requestID := ulid.MustNew(ulid.Now(), rand.Reader).String()
	ctx := r.Context()
	if s.Auditor != nil {
		s.Auditor.Record(ctx, audit.Event{
			TenantID:      firstNonEmpty(req.TenantID, r.Header.Get("X-ERA-Tenant")),
			MailboxID:     req.MailboxID,
			InferenceType: "phishing",
			Model:         "rule-based",
			RiskScore:     res.RiskScore,
			LatencyMs:     latencyMs,
			RequestID:     requestID,
			BodyHash:      audit.BodyHash(req.Message.Body),
		})
	}
	writeJSON(w, http.StatusOK, res)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func concatThread(thread []summary.Message) string {
	var b strings.Builder
	for _, m := range thread {
		b.WriteString(m.From)
		b.WriteString(m.Subject)
		b.WriteString(m.Body)
	}
	return b.String()
}

// SummarySLAMs returns the SLA threshold used in acceptance tests.
func SummarySLAMs() int64 { return summarySLAMs }
