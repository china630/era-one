package sync_test

import (
	"os"
	"testing"

	"era/services/comms/internal/imapclient"
	"era/services/comms/mail-connect/internal/sync"
)

func TestResolvePassword(t *testing.T) {
	os.Setenv("ERA_CONNECT_SECRET_ALICE", "s3cret")
	defer os.Unsetenv("ERA_CONNECT_SECRET_ALICE")
	got, err := sync.ResolvePassword("vault://alice")
	if err != nil || got != "s3cret" {
		t.Fatalf("%v %q", err, got)
	}
}

func TestStartSync_RealIMAP(t *testing.T) {
	addr, stop, err := imapclient.StartTestServer(map[string][]byte{
		"1": []byte("Subject: hi\r\n\r\nbody-one"),
		"2": []byte("Subject: bye\r\n\r\nbody-two"),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer stop()

	os.Setenv("ERA_CONNECT_SECRET_LAB", "pass")
	defer os.Unsetenv("ERA_CONNECT_SECRET_LAB")

	st := sync.NewStore()
	st.PutMailbox(sync.Mailbox{
		TenantID: "t1", Email: "a@x.local", Provider: "imap",
		Address: addr, Username: "u", PasswordRef: "vault://lab",
	})
	j := st.StartSync("t1", "a@x.local")
	if j.Status != "done" || j.ItemsOK != 2 {
		t.Fatalf("%+v", j)
	}
}

func TestStartSync_StubWithoutAddress(t *testing.T) {
	st := sync.NewStore()
	st.PutMailbox(sync.Mailbox{TenantID: "t1", Email: "a@x.local"})
	j := st.StartSync("t1", "a@x.local")
	if j.Mode != "stub" || j.ItemsOK != 0 || j.Status != "done" {
		t.Fatalf("want mode=stub items_ok=0, got %+v", j)
	}
}
