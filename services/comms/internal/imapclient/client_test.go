package imapclient

import (
	"strings"
	"testing"
)

func TestFetchAndAppendRoundTrip(t *testing.T) {
	raw := []byte("From: a@test.local\r\nTo: b@test.local\r\nSubject: golden\r\n\r\nbody\r\n")
	addr, stop, err := StartTestServer(map[string][]byte{"1": raw})
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	WaitReady()

	host, portStr, _ := strings.Cut(addr, ":")
	var port int
	for _, ch := range portStr {
		port = port*10 + int(ch-'0')
	}
	cl, err := Dial(Config{Host: host, Port: port, Username: "testuser", Password: "testpass"})
	if err != nil {
		t.Fatal(err)
	}
	defer cl.Close()

	msgs, err := cl.FetchFolder("INBOX")
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Fatalf("want 1 msg got %d", len(msgs))
	}
	want := HashRaw(raw)
	if msgs[0].Hash != want {
		t.Fatalf("hash mismatch got %s want %s", msgs[0].Hash, want)
	}

	if err := cl.Append("Sent", raw, false); err != nil {
		t.Fatal(err)
	}
}

func TestHashRawGolden(t *testing.T) {
	raw := []byte("Subject: staging\r\n\r\nSMTP staging body\r\n")
	got := HashRaw(raw)
	if len(got) != 64 {
		t.Fatalf("expected sha256 hex, got %q", got)
	}
}
