package audit

import "sync"

type Event struct {
	Action   string
	TenantID string
	RoomID   string
	User     string
}

type Recorder struct {
	mu     sync.Mutex
	events []Event
}

func NewRecorder() *Recorder { return &Recorder{} }

func (r *Recorder) Record(ev Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, ev)
}

func (r *Recorder) Count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.events)
}
