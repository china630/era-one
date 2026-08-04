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
	t.Setenv("ERA_MAIL_DEV", "1")
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
	body := rec.Body.String()
	// Honesty: calendar stub does not inflate items_total / items_ok; pst smoke → mode=stub
	if !strings.Contains(body, `"items_total":2`) {
		t.Fatalf("unexpected body %s", body)
	}
	if !strings.Contains(body, `"items_ok":0`) {
		t.Fatalf("calendar stub must not inflate items_ok; body=%s", body)
	}
	if !strings.Contains(body, `"mode":"stub"`) {
		t.Fatalf("want mode=stub in %s", body)
	}
	if !strings.Contains(body, `"calendar_stub_count":1`) {
		t.Fatalf("want calendar_stub_count=1 (reported, not counted) in %s", body)
	}
	if !strings.Contains(body, `"mailbox":"alice@mail.gov.az"`) {
		t.Fatalf("want mailbox on job response (CH audit path) in %s", body)
	}
	var sawMailbox bool
	for _, ev := range a.Events() {
		if ev.Mailbox == "alice@mail.gov.az" {
			sawMailbox = true
			break
		}
	}
	if !sawMailbox {
		t.Fatal("audit events must carry mailbox for CH migration_job")
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
