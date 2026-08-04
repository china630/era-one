package hold_test

import (
	"testing"
	"time"

	"era/services/comms/mail-moderation/internal/hold"
)

func TestHold_RequireAllQuorum(t *testing.T) {
	s := hold.NewStore()
	r, err := s.Put(hold.Record{
		Sender:     "alex@company.local",
		Recipients: []string{"x@out.com"},
		Moderators: []string{"a@c.local", "b@c.local"},
		RequireAll: true,
		Raw:        []byte("x"),
	})
	if err != nil {
		t.Fatal(err)
	}
	partial, err := s.Approve(r.ID, "a@c.local")
	if err != nil {
		t.Fatal(err)
	}
	if partial.Status != hold.StatusPending {
		t.Fatalf("want still pending after first approve, got %s", partial.Status)
	}
	done, err := s.Approve(r.ID, "b@c.local")
	if err != nil {
		t.Fatal(err)
	}
	if done.Status != hold.StatusApproved {
		t.Fatalf("want approved after quorum, got %s", done.Status)
	}
	if len(done.ApprovedBy) != 2 {
		t.Fatalf("want 2 approvers, got %v", done.ApprovedBy)
	}
}

func TestHold_ApproveRejectTTL(t *testing.T) {
	s := hold.NewStore()
	now := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	s.SetClockForTest(func() time.Time { return now })

	r, err := s.Put(hold.Record{
		Sender:     "alex@company.local",
		Recipients: []string{"x@out.com"},
		Moderators: []string{"ivan@company.local", "hr@company.local"},
		Raw:        []byte("Subject: hi\r\n\r\nbody"),
		ExpiresAt:  now.Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Reject(r.ID, "ivan@company.local", ""); err == nil {
		t.Fatal("reject without comment must fail")
	}
	if _, err := s.Reject(r.ID, "ivan@company.local", "fix price"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Approve(r.ID, "hr@company.local"); err == nil {
		t.Fatal("second action must fail")
	}

	r2, err := s.Put(hold.Record{
		Sender:     "a@c.local",
		Recipients: []string{"b@out.com"},
		Moderators: []string{"m@c.local"},
		ExpiresAt:  now.Add(time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	s.SetClockForTest(func() time.Time { return now.Add(2 * time.Minute) })
	expired := s.ExpirePending(false)
	if len(expired) != 1 || expired[0].ID != r2.ID || expired[0].Status != hold.StatusExpired {
		t.Fatalf("expire: %+v", expired)
	}
}
