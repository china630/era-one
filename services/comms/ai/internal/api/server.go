package api

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"era/services/comms/ai/internal/audit"
	"era/services/comms/ai/internal/llm"
	"era/services/comms/ai/internal/phishing"
	"era/services/comms/ai/internal/summary"
	"era/services/comms/internal/httpauth"
	"era/services/platform/licensegate"

	"github.com/oklog/ulid"
)

const summarySLAMs = 5000

type Server struct {
	LLM     llm.Client
	Gate    *licensegate.Gate
	Auditor *audit.Recorder
}

func NewServer(client llm.Client, gate *licensegate.Gate, aud *audit.Recorder) *Server {
	if client == nil {
		client = llm.Heuristic{}
	}
	return &Server{LLM: client, Gate: gate, Auditor: aud}
}

func (s *Server) Register(mux *http.ServeMux) {
	devKey := "ERA_COMMS_AI_DEV"
	if os.Getenv("ERA_COMMS_AI_DEV") != "1" && os.Getenv("ERA_MAIL_DEV") == "1" {
		devKey = "ERA_MAIL_DEV"
	}
	auth := httpauth.FromEnv(devKey, "mail.user")
	mux.HandleFunc("/healthz", s.healthz)
	mux.HandleFunc("/api/v1/ai/summary", auth.Wrap(s.withLicense(s.handleSummary)))
	mux.HandleFunc("/api/v1/ai/phishing", auth.Wrap(s.withLicense(s.handlePhishing)))
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	model := "heuristic"
	if s.LLM != nil {
		model = s.LLM.ModelName()
	}
	degraded := model == "heuristic"
	mode := "ollama"
	if degraded {
		mode = "heuristic"
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":   "ok",
		"service":  "era-comms-ai",
		"model":    model,
		"mode":     mode,
		"degraded": degraded,
	})
}

func (s *Server) withLicense(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.Gate != nil && !s.Gate.Allow(licensegate.ModuleCommsAI) {
			http.Error(w, "module comms-ai not licensed", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

// tenantID binds to JWT/internal/DEV principal from httpauth.Wrap.
// JWT claims win over spoofable X-ERA-Tenant and body tenant_id.
func tenantID(r *http.Request, bodyTenant string) string {
	if p, ok := httpauth.FromContext(r.Context()); ok && p.TenantID != "" {
		if p.Mode == "jwt" || p.Mode == "internal" {
			return p.TenantID
		}
		// DEV: principal already mirrors X-ERA-Tenant (or t-demo default)
		return p.TenantID
	}
	return bodyTenant
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
	tid := tenantID(r, req.TenantID)
	if s.Auditor != nil {
		s.Auditor.Record(ctx, audit.Event{
			TenantID:      tid,
			MailboxID:     req.MailboxID,
			InferenceType: "summary",
			Model:         res.Model,
			LatencyMs:     res.LatencyMs,
			RequestID:     requestID,
			BodyHash:      audit.BodyHash(body),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"summary":    res.Summary,
		"model":      res.Model,
		"latency_ms": res.LatencyMs,
		"degraded":   res.Model == "heuristic",
		"mode":       map[bool]string{true: "heuristic", false: "ollama"}[res.Model == "heuristic"],
		"tenant_id":  tid,
	})
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
	tid := tenantID(r, req.TenantID)
	if s.Auditor != nil {
		s.Auditor.Record(ctx, audit.Event{
			TenantID:      tid,
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
