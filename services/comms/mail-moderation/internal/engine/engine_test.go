package engine_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"era/services/comms/mail-moderation/internal/audit"
	"era/services/comms/mail-moderation/internal/engine"
	"era/services/comms/mail-moderation/internal/hold"
	"era/services/comms/mail-moderation/internal/notify"
	"era/services/comms/mail-moderation/internal/policy"
	"era/services/comms/mail-moderation/internal/resolve"
)

func TestEngine_ModeratedDL(t *testing.T) {
	tok := notify.NewTokens([]byte("x"))
	rec := &notify.Recorder{}
	up := &engine.MemoryUpstream{}
	eng := &engine.Engine{
		Rules: []policy.Rule{{
			ID: "dl-finance", Priority: 1,
			Conditions: policy.Conditions{ModeratedRecipients: []string{"finance@company.local"}},
			Moderator:  policy.ModeratorSpec{Mode: policy.ModStatic, Static: []string{"cfo@company.local"}},
		}},
		Local:    []string{"company.local"},
		Groups:   engine.StaticGroups{},
		Resolve:  &resolve.Resolver{Dir: &resolve.MemoryDir{}},
		Holds:    hold.NewStore(),
		Notify:   &notify.Service{Mailer: rec, Tokens: tok, PublicBase: "http://x"},
		Audit:    &audit.Memory{},
		Upstream: up,
	}
	_, id, err := eng.ProcessRaw([]byte("Subject: budget\r\n\r\nok"), "anyone@company.local", []string{"finance@company.local"})
	if err != nil || id == "" {
		t.Fatalf("%v %s", err, id)
	}
	if err := eng.ApplyAction(id, "cfo@company.local", "approve", ""); err != nil {
		t.Fatal(err)
	}
	if len(up.Delivered) != 1 {
		t.Fatalf("want upstream deliver, got %d", len(up.Delivered))
	}
}

func TestEngine_RejectComment(t *testing.T) {
	tok := notify.NewTokens([]byte("x"))
	rec := &notify.Recorder{}
	eng := &engine.Engine{
		Rules: []policy.Rule{{
			ID: "r1", Priority: 1,
			Conditions: policy.Conditions{ExternalOnly: true},
			Moderator:  policy.ModeratorSpec{Mode: policy.ModStatic, Static: []string{"m@c.local"}},
		}},
		Local:    []string{"c.local"},
		Groups:   engine.StaticGroups{},
		Resolve:  &resolve.Resolver{Dir: &resolve.MemoryDir{}},
		Holds:    hold.NewStore(),
		Notify:   &notify.Service{Mailer: rec, Tokens: tok, PublicBase: "http://x"},
		Audit:    &audit.Memory{},
		Upstream: &engine.MemoryUpstream{},
	}
	_, id, err := eng.ProcessRaw([]byte("Subject: x\r\n\r\nbody"), "a@c.local", []string{"b@out.com"})
	if err != nil || id == "" {
		t.Fatalf("%v %s", err, id)
	}
	if err := eng.ApplyAction(id, "m@c.local", "reject", "fix price"); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, m := range rec.Sent {
		if stringsContains(m.Subject, "Rejected") {
			found = true
		}
	}
	if !found {
		t.Fatalf("no reject mail: %+v", rec.Sent)
	}
}

func TestEngine_Escalate(t *testing.T) {
	tok := notify.NewTokens([]byte("x"))
	rec := &notify.Recorder{}
	up := &engine.MemoryUpstream{}
	eng := &engine.Engine{
		Rules: []policy.Rule{
			{
				ID: "l1", Priority: 1, EscalateTo: "l2", Level: 1,
				Conditions: policy.Conditions{ExternalOnly: true},
				Moderator:  policy.ModeratorSpec{Mode: policy.ModStatic, Static: []string{"m1@c.local"}},
			},
			{
				ID: "l2", Priority: 2, Level: 2,
				Conditions: policy.Conditions{ExternalOnly: true},
				Moderator:  policy.ModeratorSpec{Mode: policy.ModStatic, Static: []string{"m2@c.local"}},
			},
		},
		Local:    []string{"c.local"},
		Groups:   engine.StaticGroups{},
		Resolve:  &resolve.Resolver{Dir: &resolve.MemoryDir{}},
		Holds:    hold.NewStore(),
		Notify:   &notify.Service{Mailer: rec, Tokens: tok, PublicBase: "http://x"},
		Audit:    &audit.Memory{},
		Upstream: up,
	}
	_, id, err := eng.ProcessRaw([]byte("Subject: x\r\n\r\nbody"), "a@c.local", []string{"b@out.com"})
	if err != nil || id == "" {
		t.Fatalf("%v %s", err, id)
	}
	if err := eng.ApplyAction(id, "m1@c.local", "approve", ""); err != nil {
		t.Fatal(err)
	}
	if len(up.Delivered) != 0 {
		t.Fatalf("should not deliver after escalate, got %d", len(up.Delivered))
	}
	pending := eng.Holds.(hold.Lister).ListPending()
	if len(pending) != 1 || pending[0].Level < 2 || pending[0].RuleID != "l2" {
		t.Fatalf("want L2 pending, got %+v", pending)
	}
	if err := eng.ApplyAction(pending[0].ID, "m2@c.local", "approve", ""); err != nil {
		t.Fatal(err)
	}
	if len(up.Delivered) != 1 {
		t.Fatalf("want deliver after L2, got %d", len(up.Delivered))
	}
}

func TestDLPHandoffStub(t *testing.T) {
	got := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got <- string(b)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	t.Setenv("ERA_MM_DLP_URL", srv.URL)
	tok := notify.NewTokens([]byte("x"))
	eng := &engine.Engine{
		Rules: []policy.Rule{{
			ID: "dlp", Priority: 1,
			Conditions: policy.Conditions{DLPTrigger: []string{"passport"}},
			Moderator:  policy.ModeratorSpec{Mode: policy.ModStatic, Static: []string{"m@c.local"}},
		}},
		Local:   []string{"c.local"},
		Groups:  engine.StaticGroups{},
		Resolve: &resolve.Resolver{Dir: &resolve.MemoryDir{}},
		Holds:   hold.NewStore(),
		Notify:  &notify.Service{Mailer: &notify.Recorder{}, Tokens: tok, PublicBase: "http://x"},
		Audit:   &audit.Memory{},
	}
	_, id, err := eng.ProcessRaw([]byte("Subject: passport copy\r\n\r\nbody"), "a@c.local", []string{"b@c.local"})
	if err != nil || id == "" {
		t.Fatalf("%v %s", err, id)
	}
	select {
	case body := <-got:
		if !stringsContains(body, id) {
			t.Fatalf("dlp body missing hold: %s", body)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("dlp handoff timeout")
	}
}

func stringsContains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || (len(sub) > 0 && len(s) > 0 && indexOf(s, sub) >= 0))
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
