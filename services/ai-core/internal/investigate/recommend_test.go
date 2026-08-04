package investigate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGoldenRecommendations(t *testing.T) {
	got := GoldenRecommendations("malicious")
	raw, err := json.MarshalIndent(got, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	raw = append(raw, '\n')
	path := filepath.Join("testdata", "recommended_actions.malicious.golden.json")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		_ = os.MkdirAll("testdata", 0o755)
		if err := os.WriteFile(path, raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var a, b any
	_ = json.Unmarshal(raw, &a)
	_ = json.Unmarshal(want, &b)
	ag, _ := json.Marshal(a)
	bg, _ := json.Marshal(b)
	if string(ag) != string(bg) {
		t.Fatalf("mismatch\ngot %s\nwant %s", ag, bg)
	}
}

func TestConfirmRejectAudit(t *testing.T) {
	st := NewDecisionStore()
	res := BuildResult(Request{DetectionID: "d1", NodeID: "n1"}, []StoryStep{
		{Summary: "powershell.exe"},
	})
	st.Put(res)
	if res.InvestigationID == "" || len(res.RecommendedActions) == 0 {
		t.Fatalf("%+v", res)
	}
	if res.Status != "completed" {
		t.Fatalf("want status completed got %q", res.Status)
	}
	if n := len(st.List()); n != 1 {
		t.Fatalf("list want 1 got %d", n)
	}
	act := res.RecommendedActions[0].ID
	st.AppendDecision(DecisionRecord{
		InvestigationID: res.InvestigationID, ActionID: act, Decision: "confirm", Actor: "analyst",
		CustodyHash: "abc",
	})
	decs := st.Decisions()
	if len(decs) != 1 || decs[0].Decision != "confirm" {
		t.Fatalf("%+v", decs)
	}
	got, _ := st.Get(res.InvestigationID)
	if got.RecommendedActions[0].Status != "accepted" {
		t.Fatalf("%s", got.RecommendedActions[0].Status)
	}
	st.AppendDecision(DecisionRecord{
		InvestigationID: res.InvestigationID, ActionID: act, Decision: "reject", Actor: "analyst",
	})
	got, _ = st.Get(res.InvestigationID)
	if got.RecommendedActions[0].Status != "rejected" {
		t.Fatalf("%s", got.RecommendedActions[0].Status)
	}
}
