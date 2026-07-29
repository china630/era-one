package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"era/services/comms/migration/internal/audit"
	"era/services/comms/migration/internal/jobs"
)

func TestCreateJobAndRerun(t *testing.T) {
	dir := t.TempDir()
	imapFile := filepath.Join(dir, "imap.txt")
	if err := os.WriteFile(imapFile, []byte("msg1\nmsg2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	j := jobs.NewStore()
	a := audit.NewRecorder()
	NewServer(j, a).Register(mux)

	create := `{"source":"imap","mailbox":"alice@mail.gov.az","imap_file":"` + strings.ReplaceAll(imapFile, `\`, `\\`) + `","archive_file":"a.pst"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/migration/jobs", strings.NewReader(create))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"items_total":4`) {
		t.Fatalf("unexpected body %s", rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/migration/rerun", strings.NewReader(`{"source_uids":["u1","u1","u2"]}`))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"new_items":2`) {
		t.Fatalf("expected dedup rerun %s", rec.Body.String())
	}
	if a.Count() < 3 {
		t.Fatalf("expected audit events, got %d", a.Count())
	}
}
