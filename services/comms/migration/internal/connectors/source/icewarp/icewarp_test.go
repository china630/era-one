package icewarp

import (
	"fmt"
	"testing"

	"era/services/comms/internal/imapclient"
	"era/services/comms/migration/internal/importers/imap"
)

func TestDiscoverMemoryIMAP(t *testing.T) {
	addr, stop, err := imapclient.StartTestServer(map[string][]byte{
		"INBOX": []byte("From: a@b\r\n\r\nx\r\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	imapclient.WaitReady()
	host, port := splitAddr(addr)
	t.Setenv("IW_PASS", "p")
	res, err := Discover(imap.NetworkConfig{Host: host, Port: port, Username: "u@iw", PasswordRef: "env:IW_PASS"})
	if err != nil {
		t.Fatal(err)
	}
	if res.TotalMessages < 1 {
		t.Fatalf("want messages got %+v", res)
	}
}

func splitAddr(addr string) (string, int) {
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			var p int
			_, _ = fmt.Sscanf(addr[i+1:], "%d", &p)
			return addr[:i], p
		}
	}
	return addr, 143
}
