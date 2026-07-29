package autodiscover

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderGolden(t *testing.T) {
	got, err := Render("alice@mail.gov.az", "/api/v1/connect/sync")
	if err != nil {
		t.Fatal(err)
	}
	wantPath := filepath.Join("testdata", "connect_alice.golden.xml")
	wantBytes, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatal(err)
	}
	if normalize(string(wantBytes)) != normalize(got) {
		t.Fatalf("connect autodiscover mismatch\n--- got ---\n%s\n--- want ---\n%s", got, string(wantBytes))
	}
}

func normalize(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.TrimSpace(s)
}
