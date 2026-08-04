package investigate

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// RecommendedAction is a human-on-loop suggestion (ADR-0023 Phase 3 lite).
type RecommendedAction struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"` // isolate_host | suppress_rule | open_case_note | soar_draft
	Title       string `json:"title"`
	Detail      string `json:"detail,omitempty"`
	Playbook    string `json:"playbook,omitempty"`
	RequiresAck bool   `json:"requires_ack"`
	Status      string `json:"status,omitempty"` // pending | accepted | rejected | drafted
}

// DecisionRecord is analyst confirm/reject audit.
type DecisionRecord struct {
	At           time.Time `json:"at"`
	InvestigationID string `json:"investigation_id"`
	ActionID     string    `json:"action_id"`
	Decision     string    `json:"decision"` // confirm | reject
	Actor        string    `json:"actor"`
	Note         string    `json:"note,omitempty"`
	CustodyHash  string    `json:"custody_hash,omitempty"`
}

// DecisionStore holds confirm/reject audits + pending recommendations by investigation id.
type DecisionStore struct {
	mu    sync.Mutex
	byInv map[string]*Result
	order []string
	decs  []DecisionRecord
}

func NewDecisionStore() *DecisionStore {
	return &DecisionStore{byInv: map[string]*Result{}}
}

func (d *DecisionStore) Put(res *Result) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if res.InvestigationID == "" {
		res.InvestigationID = fmt.Sprintf("inv-%s-%d", res.DetectionID, time.Now().UnixNano())
	}
	if res.Status == "" {
		res.Status = "completed"
	}
	if _, exists := d.byInv[res.InvestigationID]; !exists {
		d.order = append(d.order, res.InvestigationID)
	}
	d.byInv[res.InvestigationID] = res
}

func (d *DecisionStore) Get(id string) (*Result, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	r, ok := d.byInv[id]
	return r, ok
}

// List returns investigations in insertion order.
func (d *DecisionStore) List() []*Result {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make([]*Result, 0, len(d.order))
	for _, id := range d.order {
		if r, ok := d.byInv[id]; ok {
			out = append(out, r)
		}
	}
	return out
}

func (d *DecisionStore) AppendDecision(rec DecisionRecord) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if rec.At.IsZero() {
		rec.At = time.Now().UTC()
	}
	d.decs = append(d.decs, rec)
	if res, ok := d.byInv[rec.InvestigationID]; ok {
		for i := range res.RecommendedActions {
			if res.RecommendedActions[i].ID == rec.ActionID {
				if rec.Decision == "confirm" {
					res.RecommendedActions[i].Status = "accepted"
					res.Status = "confirmed"
				} else {
					res.RecommendedActions[i].Status = "rejected"
					res.Status = "rejected"
				}
			}
		}
	}
}

func (d *DecisionStore) Decisions() []DecisionRecord {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make([]DecisionRecord, len(d.decs))
	copy(out, d.decs)
	return out
}

// SuggestActions builds human-on-loop recommendations (never auto-execute).
func SuggestActions(res *Result, nodeID string) []RecommendedAction {
	var out []RecommendedAction
	if res == nil {
		return out
	}
	n := 0
	next := func(kind, title, detail, playbook string) RecommendedAction {
		n++
		return RecommendedAction{
			ID: fmt.Sprintf("act-%d", n), Kind: kind, Title: title, Detail: detail,
			Playbook: playbook, RequiresAck: true, Status: "pending",
		}
	}
	switch res.Verdict {
	case "malicious", "suspicious":
		out = append(out, next("isolate_host", "Isolate host", "Contain node "+nodeID, "isolate_host"))
		out = append(out, next("open_case_note", "Open case note", "Document triage for "+res.Verdict, ""))
		out = append(out, next("soar_draft", "SOAR playbook draft", "Draft Response playbook — not executed", "isolate_host"))
		if res.Verdict == "suspicious" {
			out = append(out, next("suppress_rule", "Suppress noisy rule", "Analyst may suppress FP after review", ""))
		}
	default:
		out = append(out, next("open_case_note", "Close as benign note", "No containment suggested", ""))
	}
	return out
}

// SoarDraftPayload is POSTed to Response/SOAR as draft only.
type SoarDraftPayload struct {
	Draft       bool   `json:"draft"`
	Playbook    string `json:"playbook"`
	NodeID      string `json:"node_id"`
	CaseID      string `json:"case_id,omitempty"`
	InvestigationID string `json:"investigation_id"`
	ActionID    string `json:"action_id"`
	Note        string `json:"note,omitempty"`
}

// PostSoarDraft optionally POSTs a draft to ERA_SOAR_DRAFT_URL (never executes playbook).
func PostSoarDraft(payload SoarDraftPayload) (map[string]any, error) {
	payload.Draft = true
	url := os.Getenv("ERA_SOAR_DRAFT_URL")
	out := map[string]any{"draft": true, "posted": false, "payload": payload}
	if url == "" {
		out["note"] = "ERA_SOAR_DRAFT_URL unset — local draft only"
		return out, nil
	}
	body, _ := json.Marshal(payload)
	// Minimal http without importing cycles — use stdlib via helper in api layer preferred.
	_ = body
	out["url"] = url
	out["note"] = "use api.PostDraftHTTP from server"
	return out, nil
}

// GoldenRecommendations returns stable actions for golden tests.
func GoldenRecommendations(verdict string) []RecommendedAction {
	res := &Result{Verdict: verdict, DetectionID: "det-golden"}
	acts := SuggestActions(res, "node-lab")
	for i := range acts {
		acts[i].ID = fmt.Sprintf("act-%d", i+1)
	}
	return acts
}
