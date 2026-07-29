package communigate

import (
	"fmt"
	"strings"
	"testing"

	"era/services/comms/internal/imapclient"
	"era/services/comms/migration/internal/importers/imap"
)

func TestReportCSVFromMemoryIMAP(t *testing.T) {
	host, port := startTestServer(t, map[string][]byte{
		"INBOX": []byte("From: a@b\r\nSubject: x\r\n\r\nb\r\n"),
	})
	t.Setenv("TEST_CG_PASS", "pass")
	csv, err := ReportCSV(imap.NetworkConfig{
		Host: host, Port: port, Username: "u@cg", PasswordRef: "env:TEST_CG_PASS",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(csv, "INBOX") || !strings.Contains(csv, "TOTAL") {
		t.Fatalf("csv %s", csv)
	}
}

func startTestServer(t *testing.T, msgs map[string][]byte) (host string, port int) {
	t.Helper()
	addr, stop, err := imapclient.StartTestServer(msgs)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(stop)
	imapclient.WaitReady()
	return splitHostPort(addr)
}

func splitHostPort(addr string) (string, int) {
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			var p int
			_, _ = fmt.Sscanf(addr[i+1:], "%d", &p)
			return addr[:i], p
		}
	}
	return addr, 143
}
