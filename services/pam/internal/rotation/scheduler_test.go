package rotation

import (
	"testing"
	"time"

	"era/services/pam/internal/checkout"
	"era/services/pam/internal/kms"
	"era/services/pam/internal/vault"
)

func TestSchedulerRotatesAndRevokesCheckout(t *testing.T) {
	k := kms.NewSoftwareSealed()
	v := vault.New(k)
	_ = v.Unseal(make([]byte, 32))
	meta, err := v.PutStatic("t1", "db", "db.local", "admin", "old-pass")
	if err != nil {
		t.Fatal(err)
	}
	ch := checkout.NewStore()
	req, err := ch.Create("t1", meta.ID, "alice", 60, true)
	if err != nil {
		t.Fatal(err)
	}
	s := NewScheduler(v, ch, time.Millisecond)
	s.Policy.Interval = 0
	if n := s.TickOnce(); n != 1 {
		t.Fatalf("rotated=%d", n)
	}
	_, pw, err := v.Reveal(meta.ID)
	if err != nil {
		t.Fatal(err)
	}
	if pw == "old-pass" {
		t.Fatal("password not rotated")
	}
	got, ok := ch.Get(req.ID)
	if !ok || got.Status != checkout.StatusRevoked {
		t.Fatalf("checkout status=%v", got.Status)
	}
}
