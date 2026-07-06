package risk

import (
	"testing"
	"time"
)

func TestDedupSuppressesRepeat(t *testing.T) {
	s := New(10 * time.Minute)
	at := time.Now().UTC()
	if !s.ShouldEmit("r1", "node-a", at) {
		t.Fatal("first should emit")
	}
	if s.ShouldEmit("r1", "node-a", at.Add(time.Second)) {
		t.Fatal("duplicate should suppress")
	}
	if !s.ShouldEmit("r1", "node-b", at) {
		t.Fatal("other node should emit")
	}
}

func TestRiskBump(t *testing.T) {
	s := New(time.Minute)
	at := time.Now().UTC()
	s.Bump("n1", "critical", at)
	s.Bump("n1", "high", at)
	if got := s.Score("n1"); got != 40 {
		t.Fatalf("score=%v want 40", got)
	}
}
