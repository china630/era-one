package ingest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	erav1 "era/contracts/gen/era/v1"
)

func TestGoldenPIIReject(t *testing.T) {
	env := &erav1.Envelope{
		SchemaVersion: SupportedSchemaVersion,
		EventId:       []byte("0123456789abcdef01234567"),
		Category:      erav1.EventCategory_EVENT_CATEGORY_PROCESS,
		PiiSanitized:  false,
		Source: &erav1.Source{
			TenantId: "tenant-pii-test",
			NodeId:   "node-pii-test",
		},
		Payload: &erav1.Envelope_Process{
			Process: &erav1.ProcessEvent{
				CommandLine: "user@example.com secret-token",
				User:        "admin@corp.local",
			},
		},
	}
	_, err := ValidateAndEnrich(env, time.Unix(1_700_000_000, 0))
	if err == nil {
		t.Fatal("expected PII rejection")
	}
	got := strings.TrimSpace(err.Error())
	want := strings.TrimSpace(loadGolden(t, "pii_reject.golden.txt"))
	if got != want {
		t.Fatalf("golden mismatch\ngot:  %q\nwant: %q\n(run with intentional change only)", got, want)
	}
}

func loadGolden(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join("testdata", name)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v", name, err)
	}
	return string(b)
}
