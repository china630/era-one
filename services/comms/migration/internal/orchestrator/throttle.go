package orchestrator

import (
	"sync"
	"time"
)

// HostThrottle limits operations per source host key.
type HostThrottle struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	buckets map[string][]time.Time
}

func NewHostThrottle(limit int, window time.Duration) *HostThrottle {
	return &HostThrottle{limit: limit, window: window, buckets: make(map[string][]time.Time)}
}

func (t *HostThrottle) Allow(host string) {
	for {
		t.mu.Lock()
		now := time.Now()
		cut := now.Add(-t.window)
		var kept []time.Time
		for _, ts := range t.buckets[host] {
			if ts.After(cut) {
				kept = append(kept, ts)
			}
		}
		if len(kept) < t.limit {
			kept = append(kept, now)
			t.buckets[host] = kept
			t.mu.Unlock()
			return
		}
		sleep := kept[0].Add(t.window).Sub(now)
		t.mu.Unlock()
		if sleep > 0 {
			time.Sleep(sleep)
		}
	}
}
