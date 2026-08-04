// Package suppress — tenant FP suppressions from control-plane (ADR-0022 DC-02).
package suppress

import (
	"sync"
	"time"

	"era/services/platform/cpclient"
)

// Entry mirrors CP suppression.
type Entry struct {
	TenantID string
	RuleID   string
	NodeID   string
}

// Cache holds suppressions for emit-time checks.
type Cache struct {
	mu      sync.RWMutex
	entries []Entry
	CP      *cpclient.Client
}

func New(cp *cpclient.Client) *Cache {
	return &Cache{CP: cp}
}

func (c *Cache) Set(entries []Entry) {
	c.mu.Lock()
	c.entries = append([]Entry(nil), entries...)
	c.mu.Unlock()
}

func (c *Cache) Matches(tenantID, ruleID, nodeID string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, e := range c.entries {
		if e.TenantID != "" && tenantID != "" && e.TenantID != tenantID {
			continue
		}
		if e.RuleID != ruleID {
			continue
		}
		if e.NodeID == "" || e.NodeID == nodeID {
			return true
		}
	}
	return false
}

// Refresh pulls from control-plane (best-effort).
func (c *Cache) Refresh() {
	if c == nil || c.CP == nil {
		return
	}
	list, err := c.CP.ListSuppressions()
	if err != nil {
		return
	}
	entries := make([]Entry, 0, len(list))
	for _, s := range list {
		entries = append(entries, Entry{TenantID: s.TenantID, RuleID: s.RuleID, NodeID: s.NodeID})
	}
	c.Set(entries)
}

// StartPoll refreshes every interval until stop.
func (c *Cache) StartPoll(interval time.Duration, stop <-chan struct{}) {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	c.Refresh()
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			c.Refresh()
		}
	}
}
