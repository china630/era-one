package apply

import (
	"sync"
	"time"

	"era/services/ngfw/internal/policy"
)

// Attempt is one host-apply try (success, noop, or error).
type Attempt struct {
	At       time.Time   `json:"at"`
	Backend  string      `json:"backend"`
	Enabled  bool        `json:"enabled"`
	Applied  bool        `json:"applied"`
	RuleID   string      `json:"rule_id,omitempty"`
	DryRun   string      `json:"dry_run,omitempty"`
	Error    string      `json:"error,omitempty"`
	Note     string      `json:"note,omitempty"`
	Rule     policy.Rule `json:"rule,omitempty"`
}

// History is an in-memory ring of apply attempts.
type History struct {
	mu   sync.Mutex
	ring []Attempt
	max  int
}

func NewHistory(max int) *History {
	if max <= 0 {
		max = 64
	}
	return &History{max: max}
}

func (h *History) Record(a Attempt) {
	if h == nil {
		return
	}
	if a.At.IsZero() {
		a.At = time.Now().UTC()
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.ring = append(h.ring, a)
	if len(h.ring) > h.max {
		h.ring = h.ring[len(h.ring)-h.max:]
	}
}

func (h *History) Recent(n int) []Attempt {
	if h == nil {
		return nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if n <= 0 || n > len(h.ring) {
		n = len(h.ring)
	}
	out := make([]Attempt, n)
	copy(out, h.ring[len(h.ring)-n:])
	return out
}
