package orchestrator

import (
	"strings"
)

// ShardKey returns mailbox prefix bucket for worker farm sharding (Phase 4).
func ShardKey(mailbox string, shards int) int {
	if shards <= 1 {
		return 0
	}
	local := mailbox
	if i := strings.Index(mailbox, "@"); i >= 0 {
		local = mailbox[:i]
	}
	if local == "" {
		return 0
	}
	var sum int
	for _, c := range local {
		sum += int(c)
	}
	return sum % shards
}

// MatchesShard returns true if mailbox belongs to worker shard index.
func MatchesShard(mailbox string, shard, totalShards int) bool {
	return ShardKey(mailbox, totalShards) == shard
}
