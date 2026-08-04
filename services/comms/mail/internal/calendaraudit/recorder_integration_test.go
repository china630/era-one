//go:build integration

package calendaraudit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"era/services/comms/calendar/caldav"
	"era/services/comms/calendar/store"
	"era/services/comms/mail/internal/audit"
)

func chAddr() string {
	if a := os.Getenv("ERA_CH_ADDR"); a != "" {
		return a
	}
	return "127.0.0.1:9000"
}

func TestCalDAVAuditClickHouseE2E(t *testing.T) {
	w, err := audit.New(chAddr())
	if err != nil {
		t.Skipf("clickhouse unavailable: %v", err)
	}
	defer w.Close()

	ctx := context.Background()
	if err := w.ApplyMailAuditDDL(ctx); err != nil {
		t.Fatal(err)
	}

	st := store.New()
	h := &caldav.Handler{Store: st, Auditor: &Recorder{Writer: w}}

	body := `BEGIN:VEVENT
UID:audit-evt-1
SUMMARY:Audit Test
DTSTART:20260707T090000Z
END:VEVENT`
	req := httptest.NewRequest(http.MethodPut, "/caldav/alice@mail.gov.az/audit-evt-1.ics", strings.NewReader(body))
	h.ServeHTTP(httptest.NewRecorder(), req)

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		n, err := w.CountCalendarCreates(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if n > 0 {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("expected calendar audit row")
}
