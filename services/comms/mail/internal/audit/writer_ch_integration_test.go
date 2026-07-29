//go:build integration

package audit

import (
	"context"
	"crypto/rand"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	erav1 "era/contracts/gen/era/v1"
	"github.com/oklog/ulid"
)

func chAddr() string {
	if a := os.Getenv("ERA_CH_ADDR"); a != "" {
		return a
	}
	return "127.0.0.1:9000"
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "..", ".."))
	ddl := filepath.Join(root, "deploy", "clickhouse", "004_comms_mail_audit.sql")
	if _, err := os.Stat(ddl); err != nil {
		t.Fatalf("repo root / DDL not found at %s: %v", ddl, err)
	}
	return root
}

func applyCommsDDL(t *testing.T, w *Writer) {
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
		if err := w.conn.Exec(ctx, stmt); err != nil {
			t.Fatalf("ddl exec: %v\nstmt: %s", err, stmt)
		}
	}
}

// TestClickHouseInsertE2E — F-C4 / AC-C7: insert → row count > 0 (PRD §5).
func TestClickHouseInsertE2E(t *testing.T) {
	w, err := New(chAddr())
	if err != nil {
		t.Skipf("clickhouse unavailable at %s: %v", chAddr(), err)
	}
	defer w.Close()

	applyCommsDDL(t, w)

	eventID := "cm1-e2e-" + ulid.MustNew(ulid.Now(), rand.Reader).String()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := w.Insert(ctx, &erav1.MailAuditEvent{
		EventId:  eventID,
		TenantId: "tenant-e2e",
		Mailbox:  "bob@mail.gov.az",
		Action:   erav1.MailAuditAction_MAIL_AUDIT_SEND,
	}); err != nil {
		t.Fatal(err)
	}

	n, err := w.CountByEventID(ctx, eventID)
	if err != nil {
		t.Fatal(err)
	}
	if n < 1 {
		t.Fatalf("expected count >= 1 for event_id=%s, got %d", eventID, n)
	}
}
