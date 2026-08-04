//go:build integration

package auditch

import (
	"context"
	"os"
	"testing"
	"time"

	erav1 "era/contracts/gen/era/v1"
)

func TestChatVCSAuditCH(t *testing.T) {
	addr := os.Getenv("ERA_CH_ADDR")
	if addr == "" {
		t.Skip("ERA_CH_ADDR not set")
	}
	w, err := New(addr)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := w.ApplyDDL(ctx); err != nil {
		t.Fatal(err)
	}
	if err := w.RecordChatMessage(ctx, "t-demo", "room-1", "alice"); err != nil {
		t.Fatal(err)
	}
	if err := w.RecordVCSJoin(ctx, "t-demo", "vcs-1", "bob"); err != nil {
		t.Fatal(err)
	}
	nChat, err := w.CountByAction(ctx, erav1.MailAuditAction_MAIL_AUDIT_CHAT_MESSAGE.String())
	if err != nil || nChat < 1 {
		t.Fatalf("chat audit rows=%d err=%v", nChat, err)
	}
	nVCS, err := w.CountByAction(ctx, erav1.MailAuditAction_MAIL_AUDIT_VCS_ROOM_JOIN.String())
	if err != nil || nVCS < 1 {
		t.Fatalf("vcs audit rows=%d err=%v", nVCS, err)
	}
}
