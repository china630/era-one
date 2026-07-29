package imapclient

import (
	"strings"
	"testing"
)

func TestFetchFlagsSeen(t *testing.T) {
	rawUnread := []byte("From: a@test.local\r\nSubject: unread\r\n\r\nu\r\n")
	rawRead := []byte("From: a@test.local\r\nSubject: read\r\n\r\nr\r\n")
	addr, stop, err := StartTestServerFolders(map[string][]SeedMessage{
		"INBOX": {
			{Raw: rawUnread, Seen: false},
			{Raw: rawRead, Seen: true},
		},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	WaitReady()

	cl := dialTest(t, addr)
	defer cl.Close()

	msgs, err := cl.FetchFolder("INBOX")
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 {
		t.Fatalf("want 2 msgs got %d", len(msgs))
	}
	if msgs[0].Seen {
		t.Fatal("first message should be unread")
	}
	if !msgs[1].Seen {
		t.Fatal("second message should be seen")
	}
}

func TestAppendSeenFlag(t *testing.T) {
	raw := []byte("From: a@test.local\r\nSubject: seen-append\r\n\r\nbody\r\n")
	addr, stop, err := StartTestServerFolders(map[string][]SeedMessage{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	WaitReady()

	cl := dialTest(t, addr)
	defer cl.Close()

	if err := cl.Append("INBOX", raw, true); err != nil {
		t.Fatal(err)
	}
	msgs, err := cl.FetchFolder("INBOX")
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 || !msgs[0].Seen {
		t.Fatalf("append with \\Seen: got %+v", msgs)
	}
}

func TestListMailboxesDetailedSpecialUse(t *testing.T) {
	addr, stop, err := StartTestServerFolders(
		map[string][]SeedMessage{"INBOX": {}, "Sent Items": {}},
		map[string][]string{
			"Sent Items": {`\Sent`},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	WaitReady()

	cl := dialTest(t, addr)
	defer cl.Close()

	mboxes, err := cl.ListMailboxesDetailed()
	if err != nil {
		t.Fatal(err)
	}
	var sent mailboxFound
	for _, mb := range mboxes {
		if mb.Name == "Sent Items" {
			sent = mailboxFound{mb, true}
		}
	}
	if !sent.ok || !sent.mb.HasAttribute(`\Sent`) {
		t.Fatalf("missing \\Sent on Sent Items: %+v", mboxes)
	}
}

type mailboxFound struct {
	mb Mailbox
	ok bool
}

func dialTest(t *testing.T, addr string) *Client {
	t.Helper()
	host, portStr, _ := strings.Cut(addr, ":")
	var port int
	for _, ch := range portStr {
		port = port*10 + int(ch-'0')
	}
	cl, err := Dial(Config{Host: host, Port: port, Username: "testuser", Password: "testpass"})
	if err != nil {
		t.Fatal(err)
	}
	return cl
}
