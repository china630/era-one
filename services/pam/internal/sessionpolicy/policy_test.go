package sessionpolicy

import (
	"testing"
	"time"
)

func TestSweepMaxAndIdle(t *testing.T) {
	var got []string
	tr := NewTracker(Policy{MaxDuration: time.Hour, IdleTimeout: 10 * time.Minute}, func(id, reason string) {
		got = append(got, id+":"+reason)
	})
	tr.Start("s1")
	tr.Start("s2")
	tr.Touch("s2")
	now := time.Now().UTC().Add(2 * time.Hour)
	expired := tr.Sweep(now)
	if len(expired) < 1 {
		t.Fatalf("expired=%v got=%v", expired, got)
	}
}
