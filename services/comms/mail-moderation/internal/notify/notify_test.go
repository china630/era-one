package notify_test

import (
	"testing"

	"era/services/comms/mail-moderation/internal/notify"
)

func TestActionToken_OneTime(t *testing.T) {
	tok := notify.NewTokens([]byte("secret"))
	token, err := tok.Issue("hold1", "ivan@c.local", "approve")
	if err != nil {
		t.Fatal(err)
	}
	h, m, a, err := tok.Consume(token)
	if err != nil || h != "hold1" || m != "ivan@c.local" || a != "approve" {
		t.Fatalf("got %s %s %s %v", h, m, a, err)
	}
	if _, _, _, err := tok.Consume(token); err == nil {
		t.Fatal("reuse must fail")
	}
}

func TestNotifyModerator_Links(t *testing.T) {
	rec := &notify.Recorder{}
	svc := &notify.Service{
		From:       "mm@c.local",
		PublicBase: "https://mm.lab.local",
		Mailer:     rec,
		Tokens:     notify.NewTokens([]byte("s")),
	}
	if err := svc.NotifyModerator("hid", "alex@c.local", "Offer", []string{"ivan@c.local"}, "body"); err != nil {
		t.Fatal(err)
	}
	if len(rec.Sent) != 1 || rec.Sent[0].To[0] != "ivan@c.local" {
		t.Fatalf("%+v", rec.Sent)
	}
	if !contains(rec.Sent[0].Body, "/v1/moderation/action?token=") {
		t.Fatalf("missing action link: %s", rec.Sent[0].Body)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || (len(s) > 0 && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()))
}
