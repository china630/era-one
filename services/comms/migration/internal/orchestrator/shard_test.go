package orchestrator

import "testing"

func TestShardKeyDistribution(t *testing.T) {
	shards := 8
	seen := make(map[int]bool)
	for i := 0; i < 100; i++ {
		mb := "user" + string(rune('a'+i%26)) + "@lab.local"
		k := ShardKey(mb, shards)
		if k < 0 || k >= shards {
			t.Fatalf("shard %d out of range for %s", k, mb)
		}
		seen[k] = true
	}
	if len(seen) < 2 {
		t.Fatal("expected spread across shards")
	}
}
