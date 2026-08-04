package api

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"era/services/ngfw/internal/apply"
	"era/services/ngfw/internal/policy"
	"era/services/platform/envelope"
	"era/services/platform/licensegate"
)

type Server struct {
	Engine  *policy.Engine
	Pub     *envelope.Publisher
	Gate    *licensegate.Gate
	Apply   apply.Backend
	History *apply.History
}

func New(eng *policy.Engine, pub *envelope.Publisher, gate *licensegate.Gate) *Server {
	return &Server{Engine: eng, Pub: pub, Gate: gate, Apply: apply.Select(), History: apply.NewHistory(64)}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		mode := "policy_api"
		if apply.Enabled() {
			mode = "policy_api+host_apply"
		}
		backend := "noop"
		if s.Apply != nil {
			backend = s.Apply.Name()
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "ok", "mode": mode, "apply_backend": backend,
			"apply_enabled": apply.Enabled(),
		})
	})
	mux.HandleFunc("/api/v1/ngfw/evaluate", s.handleEvaluate)
	mux.HandleFunc("/api/v1/ngfw/policies", s.handlePolicies)
	mux.HandleFunc("/api/v1/ngfw/policies/", s.handlePolicyByRef)
	mux.HandleFunc("/api/v1/ngfw/apply", s.handleApply)
	mux.HandleFunc("/api/v1/ngfw/apply/history", s.handleApplyHistory)
	return mux
}

func (s *Server) requirePerimeter(w http.ResponseWriter) bool {
	if s.Gate != nil && !s.Gate.Allow(licensegate.ModulePerimeter) {
		http.Error(w, `{"error":"module perimeter not licensed"}`, http.StatusForbidden)
		return false
	}
	return true
}

func (s *Server) handlePolicies(w http.ResponseWriter, r *http.Request) {
	if !s.requirePerimeter(w) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{"engine": "era-ngfw", "mode": "policy_decision", "rules": s.Engine.List()})
	case http.MethodPost:
		var rules []policy.Rule
		if err := json.NewDecoder(r.Body).Decode(&rules); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if err := s.Engine.Replace(rules); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"saved": true, "count": len(rules)})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handlePolicyByRef(w http.ResponseWriter, r *http.Request) {
	if !s.requirePerimeter(w) {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ref := strings.TrimPrefix(r.URL.Path, "/api/v1/ngfw/policies/")
	if ref == "" || strings.Contains(ref, "/") {
		http.Error(w, "id or index required", http.StatusBadRequest)
		return
	}
	if idx, err := strconv.Atoi(ref); err == nil {
		rule, ok := s.Engine.GetByIndex(idx)
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, rule)
		return
	}
	rule, ok := s.Engine.Get(ref)
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, rule)
}

func (s *Server) handleEvaluate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requirePerimeter(w) {
		return
	}
	var flow policy.Flow
	if err := json.NewDecoder(r.Body).Decode(&flow); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	dec := s.Engine.Evaluate(flow.SrcIP, flow.DstIP, flow.Protocol, flow.DstPort)
	if s.Pub != nil {
		dir := "outbound"
		if !dec.Allowed {
			dir = "blocked"
		}
		_ = s.Pub.PublishNetwork(r.Context(), flow.SrcIP, flow.DstIP, flow.Protocol, dir, flow.DstPort)
	}
	applied := false
	dry := apply.DryRun(policy.Rule{DstPort: flow.DstPort, SrcCIDR: flow.SrcIP, DstCIDR: flow.DstIP, Protocol: flow.Protocol})
	if !dec.Allowed && apply.Enabled() && s.Apply != nil {
		rule := policy.Rule{ID: dec.RuleID, Action: policy.ActionDeny, DstPort: flow.DstPort, Protocol: flow.Protocol}
		if err := s.Apply.ApplyDeny(rule); err == nil {
			applied = true
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"decision": dec, "flow": flow.String(),
		"host_apply": applied, "dry_run": dry, "apply_env": os.Getenv("ERA_NGFW_APPLY") == "1",
	})
}

func (s *Server) handleApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requirePerimeter(w) {
		return
	}
	var rule policy.Rule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if rule.Action == "" {
		rule.Action = policy.ActionDeny
	}
	backend := "noop"
	if s.Apply != nil {
		backend = s.Apply.Name()
	}
	attempt := apply.Attempt{
		At: time.Now().UTC(), Backend: backend, Enabled: apply.Enabled(),
		RuleID: rule.ID, DryRun: apply.DryRun(rule), Rule: rule,
	}
	out := map[string]any{
		"enabled": apply.Enabled(),
		"backend": backend,
		"dry_run": attempt.DryRun,
	}
	if !apply.Enabled() {
		attempt.Applied = false
		attempt.Note = "set ERA_NGFW_APPLY=1 on Linux lab to apply nftables/iptables"
		s.recordApply(attempt)
		out["applied"] = false
		out["note"] = attempt.Note
		writeJSON(w, http.StatusOK, out)
		return
	}
	if err := s.Apply.ApplyDeny(rule); err != nil {
		attempt.Error = err.Error()
		s.recordApply(attempt)
		out["error"] = err.Error()
		writeJSON(w, http.StatusBadGateway, out)
		return
	}
	attempt.Applied = true
	s.recordApply(attempt)
	out["applied"] = true
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleApplyHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.requirePerimeter(w) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"history": s.History.Recent(50)})
}

func (s *Server) recordApply(a apply.Attempt) {
	if s.History == nil {
		s.History = apply.NewHistory(64)
	}
	s.History.Record(a)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
