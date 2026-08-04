// Package caladapter bridges unified repo to CalDAV store.Backend.
package caladapter

import (
	"era/services/comms/calendar/store"
	"era/services/comms/mail/internal/repo"
)

// Adapter wraps repo.Backend for CalDAV.
type Adapter struct {
	R repo.Backend
}

// New returns CalDAV-compatible backend backed by repo.
func New(r repo.Backend) *Adapter {
	return &Adapter{R: r}
}

func (a *Adapter) Put(owner, uid, body string) *store.Event {
	ev, err := a.R.PutCalendarEvent(owner, uid, body)
	if err != nil {
		return nil
	}
	return &store.Event{
		UID:       ev.UID,
		Owner:     ev.Owner,
		Body:      ev.Body,
		UpdatedAt: ev.UpdatedAt,
	}
}

func (a *Adapter) Get(owner, uid string) (*store.Event, bool) {
	ev, ok := a.R.GetCalendarEvent(owner, uid)
	if !ok {
		return nil, false
	}
	return &store.Event{
		UID:       ev.UID,
		Owner:     ev.Owner,
		Body:      ev.Body,
		UpdatedAt: ev.UpdatedAt,
	}, true
}

func (a *Adapter) List(owner string) []*store.Event {
	events, err := a.R.ListCalendarEvents(owner)
	if err != nil {
		return nil
	}
	out := make([]*store.Event, 0, len(events))
	for _, ev := range events {
		out = append(out, &store.Event{
			UID:       ev.UID,
			Owner:     ev.Owner,
			Body:      ev.Body,
			UpdatedAt: ev.UpdatedAt,
		})
	}
	return out
}

func (a *Adapter) FindByUID(uid string) (*store.Event, bool) {
	ev, ok := a.R.FindCalendarEventByUID(uid)
	if !ok {
		return nil, false
	}
	return &store.Event{
		UID:       ev.UID,
		Owner:     ev.Owner,
		Body:      ev.Body,
		UpdatedAt: ev.UpdatedAt,
	}, true
}
