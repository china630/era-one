package worker

import (
	"context"
	"strings"
	"testing"
	"time"

	"era/services/comms/internal/imapclient"
	"era/services/comms/migration/internal/audit"
	"era/services/comms/migration/internal/importers/imap"
	"era/services/comms/migration/internal/jobs"
)

func TestNetworkJobAllFolders(t *testing.T) {
	inbox := []byte("From: a@test.local\r\nSubject: inbox\r\n\r\ninbox\r\n")
	sent := []byte("From: a@test.local\r\nSubject: sent\r\n\r\nsent\r\n")
	addr, stop, err := imapclient.StartTestServerFolders(map[string][]imapclient.SeedMessage{
		"INBOX": {{Raw: inbox, Seen: true}},
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

	store := jobs.NewStore()
	rec := audit.NewRecorder()
	r := &Runner{Jobs: store, Audit: rec}
	mock := &MockTarget{}
	j, err := r.Start(context.Background(), JobRequest{
		Source:  "communigate",
		Mailbox: "user@test.local",
		SourceIMAP: imap.NetworkConfig{
			Host:        host,
			Port:        port,
			Username:    "testuser",
			PasswordRef: "env:ERA_TEST_IMAP_PASS",
		},
		Target:     mock,
		AllFolders: true,
		Mode:       "bulk",
	})
	if err != nil {
		t.Fatal(err)
	}
	waitJobDone(t, store, j.ID)
	if len(mock.Written) != 2 {
		t.Fatalf("want 2 written got %d", len(mock.Written))
	}
	if !mock.Written[0].Seen {
		t.Fatal("expected \\Seen preserved on first message")
	}
}

func waitJobDone(t *testing.T, store jobs.Repository, id string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		got, ok := store.Get(id)
		if ok && got.Status == "done" {
			return
		}
		if time.Now().After(deadline) {
			status := "unknown"
			if ok {
				status = got.Status
			}
			t.Fatalf("job timeout status=%s", status)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func TestNetworkJobToMockTarget(t *testing.T) {
	raw := []byte("From: a@test.local\r\nTo: b@test.local\r\nSubject: net\r\n\r\nhello\r\n")
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

	store := jobs.NewStore()
	rec := audit.NewRecorder()
	r := &Runner{Jobs: store, Audit: rec}
	mock := &MockTarget{}
	j, err := r.Start(context.Background(), JobRequest{
		Source:  "generic-imap",
		Mailbox: "user@test.local",
		SourceIMAP: imap.NetworkConfig{
			Host:        host,
			Port:        port,
			Username:    "testuser",
			PasswordRef: "env:ERA_TEST_IMAP_PASS",
		},
		Target: mock,
		Folder: "INBOX",
		Mode:   "bulk",
	})
	if err != nil {
		t.Fatal(err)
	}
	waitJobDone(t, store, j.ID)
	if len(mock.Written) != 1 {
		t.Fatalf("want 1 written got %d", len(mock.Written))
	}
}

func TestRerunDedup(t *testing.T) {
	store := jobs.NewStore()
	newItems := store.Rerun([]string{"u1", "u1", "u2"})
	if newItems != 2 {
		t.Fatalf("expected 2 new items got %d", newItems)
	}
}
