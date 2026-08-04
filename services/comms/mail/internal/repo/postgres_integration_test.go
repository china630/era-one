//go:build integration

package repo_test

import (
	"os"
	"testing"

	"era/services/comms/mail/internal/repo"
)

func pgDSN() string {
	if d := os.Getenv("ERA_COMMS_DATABASE_URL"); d != "" {
		return d
	}
	return "postgres://era:era_ci_pw@127.0.0.1:5432/era_cp?sslmode=disable"
}

func TestPostgresRestartPersistence(t *testing.T) {
	dsn := pgDSN()
	p1, err := repo.OpenPostgres(dsn)
	if err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}
	email := "restart-int-" + t.Name() + "@example.com"
	_, _ = p1.CreateMailbox("t-demo", email, "secret", 1<<20)
	// Isolate from prior lab runs on shared compose DB.
	_ = p1 // deliver only once below
	msg, err := p1.DeliverRaw(email, []byte("persist-body"), "a@b.c")
	if err != nil {
		t.Fatal(err)
	}
	_ = p1.Close()

	p2, err := repo.OpenPostgres(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer p2.Close()
	msgs, err := p2.ListMessages(email)
	if err != nil || len(msgs) < 1 {
		t.Fatalf("list after restart: %v len=%d", err, len(msgs))
	}
	found := false
	for _, m := range msgs {
		if m.ID == msg.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("id %d not found after restart among %d msgs", msg.ID, len(msgs))
	}
}

func TestPostgresEWSRestart(t *testing.T) {
	dsn := pgDSN()
	p1, err := repo.OpenPostgres(dsn)
	if err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}
	email := "ews-restart-" + t.Name() + "@example.com"
	_, _ = p1.CreateMailbox("t-demo", email, "pw", 1<<20)
	m, err := p1.AddEWSMessage(email, "Hi", "Body")
	if err != nil {
		t.Fatal(err)
	}
	id := formatMsgID(m.ID)
	_ = p1.Close()

	p2, err := repo.OpenPostgres(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer p2.Close()
	got, ok := p2.GetMessageByID(email, id)
	if !ok || got.Subject != "Hi" {
		t.Fatalf("get after restart ok=%v subj=%q", ok, got.Subject)
	}
}

func formatMsgID(id int64) string {
	return "msg-" + itoa(id)
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
