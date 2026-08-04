package autodiscover

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderGolden(t *testing.T) {
	t.Setenv("ERA_CONNECT_IMAP_HOST", "")
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

func TestRenderWithIMAPLab(t *testing.T) {
	got, err := RenderWithIMAP("lab1@mail.gov.az", "/api/v1/connect/sync", "dovecot-lab")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "<Server>dovecot-lab</Server>") {
		t.Fatalf("missing IMAP server: %s", got)
	}
}


func normalize(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.TrimSpace(s)
}
