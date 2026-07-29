package imap_generic

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"era/services/comms/internal/imapclient"
)

func TestFindFolderViaMemoryIMAP(t *testing.T) {
	host, port := startTestIMAP(t, map[string][]byte{
		"INBOX": []byte("From: a@b\r\nSubject: hi\r\n\r\nbody\r\n"),
	})
	t.Setenv("TEST_CG_PASS", "pass")
	be := New(Config{
		IMAPHost:    host,
		IMAPPort:    port,
		IMAPUser:    "u@cg.local",
		IMAPPassRef: "TEST_CG_PASS",
	})
	status, resp, err := be.ProxyEWS(context.Background(), "FindFolder", []byte(`<FindFolder/>`), http.Header{})
	if err != nil {
		t.Fatal(err)
	}
	if status != http.StatusOK {
		t.Fatalf("status %d", status)
	}
	if !strings.Contains(string(resp), "INBOX") {
		t.Fatalf("missing INBOX: %s", resp)
	}
}

func TestSyncFolderItemsViaMemoryIMAP(t *testing.T) {
	host, port := startTestIMAP(t, map[string][]byte{
		"INBOX": []byte("From: a@b\r\nSubject: pilot\r\n\r\nbody\r\n"),
	})
	t.Setenv("TEST_CG_PASS", "pass")
	be := New(Config{
		IMAPHost:    host,
		IMAPPort:    port,
		IMAPUser:    "u@cg.local",
		IMAPPassRef: "TEST_CG_PASS",
	})
	status, resp, err := be.ProxyEWS(context.Background(), "SyncFolderItems", []byte(`<SyncFolderItems/>`), http.Header{})
	if err != nil {
		t.Fatal(err)
	}
	if status != http.StatusOK || !strings.Contains(string(resp), "pilot") {
		t.Fatalf("resp %s err %v status %d", resp, err, status)
	}
}

func startTestIMAP(t *testing.T, msgs map[string][]byte) (host string, port int) {
	t.Helper()
	addr, stop, err := imapclient.StartTestServer(msgs)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(stop)
	imapclient.WaitReady()
	parts := strings.Split(addr, ":")
	return parts[0], atoi(parts[1])
}

func atoi(s string) int {
	var n int
	_, _ = fmt.Sscanf(s, "%d", &n)
	return n
}
