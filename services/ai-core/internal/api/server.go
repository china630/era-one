package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"

	"era/services/ai-core/internal/investigate"
	"era/services/platform/cpclient"
	"era/services/platform/custody"
	"era/services/platform/licensegate"
)

type Server struct {
	Inv      *investigate.Client
	Gate     *licensegate.Gate
	CP       *cpclient.Client
	Audit    *investigate.AuditLog
	Custody  *custody.Chain
	Decisions *investigate.DecisionStore
}

func New(inv *investigate.Client, gate *licensegate.Gate) *Server {
	return &Server{
		Inv:       inv,
		Gate:      gate,
		CP:        cpclient.New(os.Getenv("ERA_CONTROL_PLANE_URL")).WithActor("ai-core"),
		Audit:     investigate.NewAuditLog(),
		Custody:   custody.NewChain(),
		Decisions: investigate.NewDecisionStore(),
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/api/v1/investigate", s.handleInvestigate)
	mux.HandleFunc("/api/v1/investigate/audit", s.handleAudit)
	mux.HandleFunc("/api/v1/investigate/graph", s.handleGraph)
	mux.HandleFunc("/api/v1/investigate/", s.handleInvestigateSub)
	return mux
}

func (s *Server) handleInvestigateSub(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/investigate/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	id := parts[0]
	if len(parts) == 1 {
		s.handleInvestigateGet(w, r, id)
		return
	}
	action := parts[1]
	switch action {
	case "confirm":
		s.handleDecision(w, r, id, "confirm")
	case "reject":
		s.handleDecision(w, r, id, "reject")
	case "soar-draft":
		s.handleSoarDraft(w, r, id)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleInvestigateGet(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.Gate.Allow(licensegate.ModuleControlAI) {
		http.Error(w, "module ai not licensed", http.StatusForbidden)
		return
	}
	if s.Decisions == nil {
		http.NotFound(w, r)
		return
	}
	res, ok := s.Decisions.Get(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleInvestigate(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleInvestigateList(w, r)
		return
	case http.MethodPost:
		// continue below
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.Gate.Allow(licensegate.ModuleControlAI) {
		http.Error(w, "module ai not licensed", http.StatusForbidden)
		return
	}
	var req investigate.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	res, err := s.Inv.Investigate(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if s.CP != nil && (res.Verdict == "malicious" || res.Verdict == "suspicious") {
		title := "AI investigation: " + res.Verdict + " on " + req.NodeID
		if id, err := s.CP.CreateCase(title, req.DetectionID, req.NodeID); err == nil {
			res.CaseID = id
		}
	}
	investigate.SealEvidence(s.Custody, res)
	actor := r.Header.Get("X-ERA-Actor")
	if actor == "" {
		actor = "ai-core"
	}
	s.Audit.Append(investigate.AuditEntry{
		Actor: actor, CaseID: res.CaseID, NodeID: req.NodeID, DetectionID: req.DetectionID,
		Verdict: res.Verdict, ModelVersion: res.ModelVersion, PromptHash: res.PromptHash,
		CustodyHash: res.CustodyRootHash,
	})
	if s.Decisions != nil {
		s.Decisions.Put(res)
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleInvestigateList(w http.ResponseWriter, r *http.Request) {
	if !s.Gate.Allow(licensegate.ModuleControlAI) {
		http.Error(w, "module ai not licensed", http.StatusForbidden)
		return
	}
	list := []*investigate.Result{}
	if s.Decisions != nil {
		list = s.Decisions.List()
	}
	writeJSON(w, http.StatusOK, map[string]any{"investigations": list})
}

func (s *Server) handleDecision(w http.ResponseWriter, r *http.Request, invID, decision string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.Gate.Allow(licensegate.ModuleControlAI) {
		http.Error(w, "module ai not licensed", http.StatusForbidden)
		return
	}
	var body struct {
		ActionID string `json:"action_id"`
		Note     string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ActionID == "" {
		http.Error(w, "action_id required", http.StatusBadRequest)
		return
	}
	actor := r.Header.Get("X-ERA-Actor")
	if actor == "" {
		actor = "analyst"
	}
	custodyHash := ""
	if s.Custody != nil {
		entry := s.Custody.Seal([]byte("ai-decision:" + invID + ":" + body.ActionID + ":" + decision))
		custodyHash = entry.Hash
	}
	rec := investigate.DecisionRecord{
		InvestigationID: invID, ActionID: body.ActionID, Decision: decision,
		Actor: actor, Note: body.Note, CustodyHash: custodyHash,
	}
	if s.Decisions != nil {
		s.Decisions.AppendDecision(rec)
	}
	s.Audit.Append(investigate.AuditEntry{
		Actor: actor, DetectionID: invID, Verdict: decision + ":" + body.ActionID,
		ModelVersion: "human-on-loop", PromptHash: "", CustodyHash: custodyHash,
	})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "decision": rec})
}

func (s *Server) handleSoarDraft(w http.ResponseWriter, r *http.Request, invID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.Gate.Allow(licensegate.ModuleControlAI) {
		http.Error(w, "module ai not licensed", http.StatusForbidden)
		return
	}
	var body struct {
		ActionID string `json:"action_id"`
		Playbook string `json:"playbook"`
		NodeID   string `json:"node_id"`
		CaseID   string `json:"case_id"`
		Note     string `json:"note"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	payload := investigate.SoarDraftPayload{
		Draft: true, Playbook: body.Playbook, NodeID: body.NodeID, CaseID: body.CaseID,
		InvestigationID: invID, ActionID: body.ActionID, Note: body.Note,
	}
	if payload.Playbook == "" {
		payload.Playbook = "isolate_host"
	}
	out := map[string]any{"draft": true, "auto_execute": false, "payload": payload}
	if url := os.Getenv("ERA_SOAR_DRAFT_URL"); url != "" {
		raw, _ := json.Marshal(payload)
		resp, err := http.Post(url, "application/json", bytes.NewReader(raw))
		if err != nil {
			out["error"] = err.Error()
			writeJSON(w, http.StatusBadGateway, out)
			return
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		out["posted"] = true
		out["soar_status"] = resp.StatusCode
		out["soar_body"] = string(b)
	} else {
		out["posted"] = false
		out["note"] = "ERA_SOAR_DRAFT_URL unset — draft retained locally; not auto-executed"
	}
	s.Audit.Append(investigate.AuditEntry{
		Actor: "ai-core", DetectionID: invID, Verdict: "soar_draft",
		ModelVersion: "human-on-loop", CustodyHash: "",
	})
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleAudit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.Gate.Allow(licensegate.ModuleControlAI) {
		http.Error(w, "module ai not licensed", http.StatusForbidden)
		return
	}
	decisions := []investigate.DecisionRecord{}
	if s.Decisions != nil {
		decisions = s.Decisions.Decisions()
	}
	writeJSON(w, http.StatusOK, map[string]any{"audit": s.Audit.List(), "decisions": decisions})
}

func (s *Server) handleGraph(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.Gate.Allow(licensegate.ModuleControlAI) {
		http.Error(w, "module ai not licensed", http.StatusForbidden)
		return
	}
	var body struct {
		Storyline []investigate.StoryStep `json:"storyline"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, investigate.BuildAttackGraph(body.Storyline))
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
