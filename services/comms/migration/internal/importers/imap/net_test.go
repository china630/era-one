package imap

import (
	"strings"
	"testing"

	"era/services/comms/internal/imapclient"
)

func TestImportNetwork(t *testing.T) {
	raw := []byte("Subject: staging\r\n\r\nSMTP staging body\r\n")
	addr, stop, err := imapclient.StartTestServer(map[string][]byte{"1": raw})
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	imapclient.WaitReady()
	host, portStr, _ := strings.Cut(addr, ":")
	var port int
	for _, ch := range portStr {
		port = port*10 + int(ch-'0')
	}
	t.Setenv("ERA_TEST_IMAP_PASS", "testpass")

	msgs, err := ImportNetwork(NetworkConfig{
		Host:        host,
		Port:        port,
		Username:    "testuser",
		PasswordRef: "env:ERA_TEST_IMAP_PASS",
	}, "INBOX")
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Fatalf("want 1 got %d", len(msgs))
	}
	if msgs[0].Subject != "staging" {
		t.Fatalf("subject %q", msgs[0].Subject)
	}
}

func TestImportNetworkAll(t *testing.T) {
	inbox := []byte("Subject: inbox\r\n\r\ninbox body\r\n")
	sent := []byte("Subject: sent\r\n\r\nsent body\r\n")
	addr, stop, err := imapclient.StartTestServerFolders(map[string][]imapclient.SeedMessage{
		"INBOX": {{Raw: inbox}},
		"Sent":  {{Raw: sent}},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	imapclient.WaitReady()
	host, portStr, _ := strings.Cut(addr, ":")
	var port int
	for _, ch := range portStr {
		port = port*10 + int(ch-'0')
	}
	t.Setenv("ERA_TEST_IMAP_PASS", "testpass")

	msgs, err := ImportNetworkAll(NetworkConfig{
		Host:        host,
		Port:        port,
		Username:    "testuser",
		PasswordRef: "env:ERA_TEST_IMAP_PASS",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 {
		t.Fatalf("want 2 got %d", len(msgs))
	}
}
