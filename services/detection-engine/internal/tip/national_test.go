package tip

import "testing"

func TestNationalIOCMatch(t *testing.T) {
	f := FromPatterns([]string{"evil.az", "bad-c2.example"})
	if ok, _ := f.Match(`{"dns":"evil.az"}`); !ok {
		t.Fatal("expected match")
	}
	if ok, _ := f.Match(`{"dns":"safe.az"}`); ok {
		t.Fatal("unexpected match")
	}
}

func TestDetectionDelta(t *testing.T) {
	f := FromPatterns([]string{"evil.az"})
	baseline := 0
	withNat := 0
	if ok, _ := f.Match(`{"x":"evil.az"}`); ok {
		withNat++
	}
	if withNat <= baseline {
		t.Fatal("national feed should improve detection count")
	}
}
