//go:build integration

package auditapi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"era/services/comms/mail/internal/audit"
	"era/services/comms/mail/internal/auditapi"
)

func chAddr() string {
	if a := os.Getenv("ERA_CH_ADDR"); a != "" {
		return a
	}
	return "127.0.0.1:19000"
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "..", ".."))
}

func applyCommsDDL(t *testing.T, w *audit.Writer) {
	t.Helper()
	root := repoRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "deploy", "clickhouse", "004_comms_mail_audit.sql"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, stmt := range strings.Split(string(raw), ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || strings.HasPrefix(stmt, "--") {
			continue
		}
		if err := w.ApplyMailAuditDDL(ctx); err != nil {
			// Prefer writer helper; fall through if DDL file path used elsewhere
			_ = err
		}
		break
	}
	if err := w.ApplyMailAuditDDL(ctx); err != nil {
		t.Fatalf("ddl: %v", err)
	}
}

// TestAuditPathSMTPToClickHouse — F-C4: SMTP → audit_hook → auditapi → CH row.
func TestAuditPathSMTPToClickHouse(t *testing.T) {
	w, err := audit.New(chAddr())
	if err != nil {
		t.Skipf("clickhouse unavailable: %v", err)
	}
	defer w.Close()
	applyCommsDDL(t, w)

	srv := httptest.NewServer(&auditapi.Handler{Writer: w})
	defer srv.Close()

	root := repoRoot(t)
	cmd := exec.Command("cargo", "test", "-p", "era-mail-core", "--test", "smtp_audit_e2e", "--quiet")
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"ERA_MAIL_AUDIT_URL="+srv.URL+"/internal/v1/audit",
		"ERA_MAIL_DEFAULT_TENANT=tenant-e2e",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("cargo smtp_audit_e2e: %v\n%s", err, out)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	deadline := time.Now().Add(5 * time.Second)
	var n uint64
	for time.Now().Before(deadline) {
		n, err = w.CountSendsByMailbox(ctx, "bob@mail.gov.az")
		if err != nil {
			t.Fatal(err)
		}
		if n > 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if n < 1 {
		t.Fatalf("expected mail_audit SEND rows for bob@mail.gov.az, got %d", n)
	}
}

// TestAuditPathWebhookToClickHouse — auditapi POST → CH (handler path).
func TestAuditPathWebhookToClickHouse(t *testing.T) {
	w, err := audit.New(chAddr())
	if err != nil {
		t.Skipf("clickhouse unavailable: %v", err)
	}
	defer w.Close()
	applyCommsDDL(t, w)

	srv := httptest.NewServer(&auditapi.Handler{Writer: w})
	defer srv.Close()

	client := srv.Client()
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/internal/v1/audit",
		strings.NewReader(`{"tenant_id":"tenant-e2e","mailbox":"bob@mail.gov.az","action":"send","mail_from":"alice@mail.gov.az","src_ip":"127.0.0.1"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status %d", resp.StatusCode)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	n, err := w.CountSendsByMailbox(ctx, "bob@mail.gov.az")
	if err != nil {
		t.Fatal(err)
	}
	if n < 1 {
		t.Fatalf("expected audit row, got %d", n)
	}
}
