package investigate

import (
	"encoding/json"
	"testing"

	"era/services/platform/custody"
)

func TestBuildResultAuditCustodyGraph(t *testing.T) {
	req := Request{DetectionID: "d1", NodeID: "n1"}
	steps := []StoryStep{
		{EventID: "e3", Category: "auth", ObservedAt: "t3", Summary: "auth failed"},
		{EventID: "e2", Category: "network", ObservedAt: "t2", Summary: "10.0.0.1 → 10.0.0.2"},
		{EventID: "e1", Category: "process", ObservedAt: "t1", Summary: "powershell.exe"},
	}
	res := BuildResult(req, steps)
	if res.Verdict != "malicious" {
		t.Fatalf("verdict=%s", res.Verdict)
	}
	if res.PromptHash == "" || res.ModelVersion == "" {
		t.Fatal("missing forensic fields")
	}
	chain := custody.NewChain()
	h := SealEvidence(chain, res)
	if h == "" || res.CustodyRootHash != h {
		t.Fatalf("custody %q", h)
	}
	if chain.Head() != h {
		t.Fatalf("head %s want %s", chain.Head(), h)
	}
	g := BuildAttackGraph(steps)
	if len(g.Nodes) != 3 || len(g.Edges) != 2 {
		t.Fatalf("graph nodes=%d edges=%d", len(g.Nodes), len(g.Edges))
	}
	// edges should be process → network → auth (causal)
	if g.Edges[0].From != "e1" || g.Edges[0].To != "e2" {
		b, _ := json.Marshal(g)
		t.Fatalf("edges order: %s", b)
	}
	log := NewAuditLog()
	log.Append(AuditEntry{Actor: "a", NodeID: "n1", Verdict: res.Verdict, PromptHash: res.PromptHash, CustodyHash: h})
	if len(log.List()) != 1 {
		t.Fatal("audit")
	}
}
