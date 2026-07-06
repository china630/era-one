package report

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRegulatoryFromCHGolden(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("query")
		var resp string
		switch {
		case contains(q, "pii_sanitized = 0"):
			resp = `{"data":[{"count()":"0"}]}`
		case contains(q, "era_xdr.events") && contains(q, "count()"):
			resp = `{"data":[{"count()":"42000"}]}`
		case contains(q, "era_xdr.detections") && contains(q, "severity = 'critical'"):
			resp = `{"data":[{"count()":"2"}]}`
		case contains(q, "era_xdr.detections"):
			resp = `{"data":[{"count()":"15"}]}`
		case contains(q, "uniqExact"):
			resp = `{"data":[{"uniqExact(node_id)":"12"}]}`
		default:
			resp = `{"data":[{"count()":"0"}]}`
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(resp))
	}))
	defer srv.Close()

	ch := &CHClient{BaseURL: srv.URL, HTTPClient: srv.Client()}
	start := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	doc := DocumentFromCHWithClient(ch, "GoldenBank", start, end)

	goldenPath := filepath.Join("testdata", "regulatory_ch.golden.json")
	if *updateGolden {
		b, _ := json.MarshalIndent(doc.Summary, "", "  ")
		_ = os.MkdirAll("testdata", 0o755)
		_ = os.WriteFile(goldenPath, b, 0o644)
	}
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("golden: %v (run with -update)", err)
	}
	var wantSummary, gotSummary Summary
	if err := json.Unmarshal(want, &wantSummary); err != nil {
		t.Fatal(err)
	}
	gotSummary = doc.Summary
	if gotSummary.TotalEvents != wantSummary.TotalEvents ||
		gotSummary.Detections != wantSummary.Detections ||
		gotSummary.ComplianceStatus != wantSummary.ComplianceStatus {
		t.Fatalf("golden mismatch:\ngot=%+v\nwant=%+v", gotSummary, wantSummary)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		(len(s) > 0 && stringIndex(s, sub) >= 0))
}

func stringIndex(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

var updateGolden = func() *bool {
	f := false
	for _, a := range os.Args[1:] {
		if a == "-update" {
			f = true
		}
	}
	return &f
}()
