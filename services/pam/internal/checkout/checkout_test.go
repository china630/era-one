package checkout

import (
	"testing"
	"time"
)

func TestCheckoutTTLAndConsume(t *testing.T) {
	st := NewStore()
	r, err := st.Create("t1", "sec1", "alice", 1, true)
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != StatusApproved {
		t.Fatal("auto approve")
	}
	got, ok := st.Consume(r.ID)
	if !ok || got.Status != StatusConsumed {
		t.Fatal("consume")
	}
	_, ok = st.Consume(r.ID)
	if ok {
		t.Fatal("double consume")
	}
}

func TestCheckoutPolicy(t *testing.T) {
	if _, ok := PolicyAllow("guest"); ok {
		t.Fatal("guest denied")
	}
	auto, ok := PolicyAllow("admin")
	if !ok || !auto {
		t.Fatal("admin")
	}
}

func TestCheckoutExpiry(t *testing.T) {
	st := NewStore()
	r, _ := st.Create("t1", "s", "bob", 0, true)
	r.ExpiresAt = time.Now().UTC().Add(-time.Minute)
	st.items[r.ID] = r
	_, ok := st.Consume(r.ID)
	if ok {
		t.Fatal("expired")
	}
}
