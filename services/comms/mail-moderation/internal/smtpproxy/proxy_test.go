package smtpproxy_test

import (
	"strings"
	"testing"

	"era/services/comms/mail-moderation/internal/audit"
	"era/services/comms/mail-moderation/internal/engine"
	"era/services/comms/mail-moderation/internal/hold"
	"era/services/comms/mail-moderation/internal/notify"
	"era/services/comms/mail-moderation/internal/policy"
	"era/services/comms/mail-moderation/internal/resolve"
	"era/services/comms/mail-moderation/internal/smtpproxy"
)

func TestSMTP_HoldApproveDeliver(t *testing.T) {
	up := &engine.MemoryUpstream{}
	tok := notify.NewTokens([]byte("s"))
	rec := &notify.Recorder{}
	dir := &resolve.MemoryDir{
		Managers: map[string]string{"alex@company.local": "ivan@company.local"},
	}
	eng := &engine.Engine{
		Rules: []policy.Rule{{
			ID: "novices-external", Priority: 10, StopProcessing: true,
			Conditions: policy.Conditions{SenderGroups: []string{"novices"}, ExternalOnly: true},
			Moderator:  policy.ModeratorSpec{Mode: policy.ModManager, Fallback: []string{"hr@company.local"}},
		}},
		Local:    []string{"company.local"},
		Groups:   engine.StaticGroups{"alex@company.local": {"novices"}},
		Resolve:  &resolve.Resolver{Dir: dir},
		Holds:    hold.NewStore(),
		Notify:   &notify.Service{Mailer: rec, Tokens: tok, PublicBase: "http://127.0.0.1"},
		Audit:    &audit.Memory{},
		Upstream: up,
	}
	srv := &smtpproxy.Server{Engine: eng}
	if err := srv.Listen("127.0.0.1:0"); err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	raw := "From: alex@company.local\r\nTo: client@external.com\r\nSubject: Offer\r\n\r\nHello\r\n"
	resp, err := smtpproxy.Submit(srv.Addr().String(), "alex@company.local", []string{"client@external.com"}, raw)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(resp, "Queued for moderation") {
		t.Fatalf("resp %q", resp)
	}
	if len(up.Delivered) != 0 {
		t.Fatal("must not deliver yet")
	}
	// find hold via audit
	mem := eng.Audit.(*audit.Memory)
	if len(mem.Events) == 0 {
		t.Fatal("no audit")
	}
	holdID := mem.Events[0].HoldID
	if err := eng.ApplyAction(holdID, "ivan@company.local", "approve", ""); err != nil {
		t.Fatal(err)
	}
	if len(up.Delivered) != 1 {
		t.Fatalf("want deliver, got %d", len(up.Delivered))
	}
}
