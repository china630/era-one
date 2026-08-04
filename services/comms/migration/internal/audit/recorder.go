package audit

import "sync"

type Event struct {
	JobID     string
	Action    string
	SourceUID string
	Detail    string
	Mailbox   string
}

// Recorder stores migration audit events (memory + optional ClickHouse).
type Recorder interface {
	Record(ev Event) bool
	Count() int
}

type MemoryRecorder struct {
	mu      sync.Mutex
	events  []Event
	seenUID map[string]bool
}

func NewRecorder() *MemoryRecorder {
	return &MemoryRecorder{seenUID: make(map[string]bool)}
}

func (r *MemoryRecorder) Record(ev Event) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ev.SourceUID != "" && r.seenUID[ev.SourceUID] && ev.Action == "MIGRATION_RERUN" {
		return false
	}
	if ev.SourceUID != "" && ev.Action == "MIGRATION_RERUN" {
		r.seenUID[ev.SourceUID] = true
	}
	r.events = append(r.events, ev)
	return true
}

func (r *MemoryRecorder) Count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.events)
}

// Events returns a copy of recorded events (tests / CH mailbox honesty).
func (r *MemoryRecorder) Events() []Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Event, len(r.events))
	copy(out, r.events)
	return out
}

// Composite forwards to memory and ClickHouse.
type Composite struct {
	Mem *MemoryRecorder
	CH  *CHWriter
}

func (c *Composite) Record(ev Event) bool {
	ok := true
	if c.Mem != nil {
		ok = c.Mem.Record(ev)
	}
	if c.CH != nil {
		_ = c.CH.Record(ev)
	}
	return ok
}

func (c *Composite) Count() int {
	if c.Mem != nil {
		return c.Mem.Count()
	}
	return 0
}
