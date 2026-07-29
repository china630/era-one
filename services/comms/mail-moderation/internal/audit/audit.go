// Package audit — hold/approve/reject/expire events (AC-MM-8).
package audit

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// Event — одна запись аудита.
type Event struct {
	EventID   string
	Observed  time.Time
	HoldID    string
	Action    string // hold|approve|reject|expire
	Sender    string
	RuleID    string
	Moderator string
	Meta      map[string]string
}

// Recorder — интерфейс записи.
type Recorder interface {
	Record(e Event) error
}

// Memory — in-memory + optional fan-out.
type Memory struct {
	mu     sync.Mutex
	Events []Event
}

func (m *Memory) Record(e Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e.EventID == "" {
		e.EventID = newID()
	}
	if e.Observed.IsZero() {
		e.Observed = time.Now().UTC()
	}
	m.Events = append(m.Events, e)
	return nil
}

// Composite пишет в несколько recorder.
type Composite struct {
	Sinks []Recorder
}

func (c *Composite) Record(e Event) error {
	for _, s := range c.Sinks {
		if err := s.Record(e); err != nil {
			return err
		}
	}
	return nil
}

// LogSink — no-op CH placeholder (env wiring in main); records accepted for tests.
type LogSink struct{}

func (LogSink) Record(Event) error { return nil }

func newID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
