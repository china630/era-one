package api

import (
	"bytes"
	"encoding/json"
	"flag"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"era/services/comms/ai/internal/audit"
	"era/services/comms/ai/internal/llm"
	"era/services/comms/ai/internal/phishing"
	"era/services/comms/ai/internal/summary"
	"era/services/platform/licensegate"
)

var updateGolden = flag.Bool("update", false, "update golden files")

func aiMux() *http.ServeMux {
	gate := licensegate.FromModules([]licensegate.Module{licensegate.ModuleCommsAI})
	aud := audit.NewRecorder(nil)
	s := NewServer(llm.Heuristic{}, gate, aud)
	mux := http.NewServeMux()
	s.Register(mux)
	return mux
}

func withMailHeaders(req *http.Request) {
	req.Header.Set("X-ERA-Tenant", "t-demo")
	req.Header.Set("X-ERA-Role", "mail.user")
}

func TestF_C31_MailSummaryOnPrem(t *testing.T) {
	mux := aiMux()
	body := map[string]any{
		"tenant_id":  "t-demo",
		"mailbox_id": "mb-1",
		"thread": []summary.Message{
			{From: "alice@demo.local", Subject: "Budget", Body: "Please review the Q3 budget figures."},
			{From: "bob@demo.local", Subject: "Re: Budget", Body: "Approved with minor changes."},
		},
	}
	b, _ := json.Marshal(body)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/summary", bytes.NewReader(b))
	withMailHeaders(req)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("summary status %d body=%s", rec.Code, rec.Body.String())
	}
	var res summary.Result
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if res.Summary == "" {
		t.Fatal("empty summary")
	}
	if res.Model != "heuristic" {
		t.Fatalf("expected heuristic model, got %q", res.Model)
	}
	if res.LatencyMs >= SummarySLAMs() {
		t.Fatalf("latency %dms exceeds SLA %dms", res.LatencyMs, SummarySLAMs())
	}
}

func TestF_C32_PhishingGolden(t *testing.T) {
	cases := []string{"phishing_benign", "phishing_malicious"}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			msg := loadPhishingFixture(t, name+".json")
			got := phishing.Classify(msg)
			wantPath := filepath.Join(testdataDir(t), name+".golden.json")
			if *updateGolden {
				writeJSONFile(t, wantPath, got)
			}
			want := loadGoldenResult(t, wantPath)
			if got.RiskScore != want.RiskScore || got.Verdict != want.Verdict {
				t.Fatalf("risk/verdict got=%+v want=%+v", got, want)
			}
			if len(got.Hints) != len(want.Hints) {
				t.Fatalf("hints len got=%d want=%d: %v vs %v", len(got.Hints), len(want.Hints), got.Hints, want.Hints)
			}
			for i := range want.Hints {
				if got.Hints[i] != want.Hints[i] {
					t.Fatalf("hint[%d] got=%q want=%q", i, got.Hints[i], want.Hints[i])
				}
			}
		})
	}
}

func TestF_C32_PhishingAPIAudit(t *testing.T) {
	aud := audit.NewRecorder(nil)
	gate := licensegate.FromModules([]licensegate.Module{licensegate.ModuleCommsAI})
	s := NewServer(llm.Heuristic{}, gate, aud)
	mux := http.NewServeMux()
	s.Register(mux)

	msg := loadPhishingFixture(t, "phishing_malicious.json")
	body := map[string]any{
		"tenant_id":  "t-demo",
		"mailbox_id": "mb-1",
		"message":    msg,
	}
	b, _ := json.Marshal(body)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/phishing", bytes.NewReader(b))
	withMailHeaders(req)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("phishing status %d", rec.Code)
	}
	if aud.Count() != 1 {
		t.Fatalf("expected 1 audit event, got %d", aud.Count())
	}
}

func TestF_C34_ResourceBudget(t *testing.T) {
	// ADR-0009: comms-ai heuristic path must stay lightweight in CI.
	const maxAllocDelta = 50 * 1024 * 1024
	const maxGoroutines = 100

	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	beforeG := runtime.NumGoroutine()

	mux := aiMux()
	for i := 0; i < 200; i++ {
		body := map[string]any{
			"tenant_id":  "t-demo",
			"mailbox_id": "mb-budget",
			"thread": []summary.Message{
				{From: "a@demo.local", Subject: "t", Body: "short body"},
			},
		}
		b, _ := json.Marshal(body)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/summary", bytes.NewReader(b))
		withMailHeaders(req)
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("iteration %d status %d", i, rec.Code)
		}
	}

	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	afterG := runtime.NumGoroutine()

	if delta := int64(after.Alloc) - int64(before.Alloc); delta > maxAllocDelta {
		t.Fatalf("alloc delta %d exceeds budget %d", delta, maxAllocDelta)
	}
	if afterG-beforeG > maxGoroutines {
		t.Fatalf("goroutine delta %d exceeds budget %d", afterG-beforeG, maxGoroutines)
	}
}

func TestCommsAIRBAC(t *testing.T) {
	mux := aiMux()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/summary", bytes.NewReader([]byte(`{}`)))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden, got %d", rec.Code)
	}
}

func testdataDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata")
}

func loadPhishingFixture(t *testing.T, name string) phishing.Message {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(testdataDir(t), name))
	if err != nil {
		t.Fatal(err)
	}
	var msg phishing.Message
	if err := json.Unmarshal(b, &msg); err != nil {
		t.Fatal(err)
	}
	return msg
}

func loadGoldenResult(t *testing.T, path string) phishing.Result {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var res phishing.Result
	if err := json.Unmarshal(b, &res); err != nil {
		t.Fatal(err)
	}
	return res
}

func writeJSONFile(t *testing.T, path string, v any) {
	t.Helper()
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}
