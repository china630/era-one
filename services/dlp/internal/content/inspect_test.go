package content

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestInspectSSN(t *testing.T) {
	e := Default()
	r := e.Inspect(Request{Content: "employee SSN 123-45-6789 note"})
	if !r.Blocked || len(r.Findings) == 0 {
		t.Fatalf("%+v", r)
	}
	if r.Findings[0].RuleID != "era-dlp-ssn" {
		t.Fatalf("%s", r.Findings[0].RuleID)
	}
}

func TestInspectPathSecrets(t *testing.T) {
	e := Default()
	r := e.Inspect(Request{Path: "/var/app/db_secret.json"})
	if !r.Blocked {
		t.Fatalf("%+v", r)
	}
}

func TestGoldenInspect(t *testing.T) {
	e := Default()
	inPath := filepath.Join("testdata", "inspect_input.json")
	gotPath := filepath.Join("testdata", "inspect_result.golden.json")
	raw, err := os.ReadFile(inPath)
	if err != nil {
		t.Fatal(err)
	}
	var req Request
	if err := json.Unmarshal(raw, &req); err != nil {
		t.Fatal(err)
	}
	res := e.Inspect(req)
	got, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(gotPath, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(gotPath)
	if err != nil {
		t.Fatal(err)
	}
	var a, b any
	_ = json.Unmarshal(got, &a)
	_ = json.Unmarshal(want, &b)
	ag, _ := json.Marshal(a)
	bg, _ := json.Marshal(b)
	if string(ag) != string(bg) {
		t.Fatalf("golden mismatch:\ngot=%s\nwant=%s", got, want)
	}
}
