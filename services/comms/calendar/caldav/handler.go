// Package caldav — minimal CalDAV subset (RFC 4791 MVP, Wave C-2).
package caldav

import (
	"context"
	"io"
	"net/http"
	"strings"

	"era/services/comms/calendar/ical"
	"era/services/comms/calendar/store"
)

const (
	methodPropfind = "PROPFIND"
	methodReport   = "REPORT"
)

// Auditor records calendar events to ClickHouse (optional).
type Auditor interface {
	RecordCalendar(ctx context.Context, create bool, owner, uid string) error
}

// Handler serves CalDAV endpoints under /caldav/.
type Handler struct {
	Store   store.Backend
	Auditor Auditor
}

// ServeHTTP routes CalDAV methods.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/caldav/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 1 || parts[0] == "" {
		http.Error(w, "user required", http.StatusBadRequest)
		return
	}
	user := parts[0]
	switch r.Method {
	case http.MethodOptions:
		w.Header().Set("DAV", "1, 2, calendar-access")
		w.Header().Set("Allow", "OPTIONS, GET, PUT, PROPFIND, REPORT")
		w.WriteHeader(http.StatusOK)
	case methodPropfind:
		h.propfind(w, user)
	case http.MethodGet:
		if len(parts) < 2 {
			http.Error(w, "resource required", http.StatusNotFound)
			return
		}
		uid := strings.TrimSuffix(parts[1], ".ics")
		ev, ok := h.Store.Get(user, uid)
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
		_, _ = w.Write([]byte(ev.Body))
	case http.MethodPut:
		if len(parts) < 2 {
			http.Error(w, "resource required", http.StatusBadRequest)
			return
		}
		uid := strings.TrimSuffix(parts[1], ".ics")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		bodyStr := string(body)
		parsed, _ := ical.ParseVEVENT(bodyStr)
		if parsed.Method == "REPLY" && len(parsed.Attendees) > 0 {
			h.handleInviteReply(user, parsed)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_, existed := h.Store.Get(user, uid)
		h.Store.Put(user, uid, bodyStr)
		if h.Auditor != nil {
			_ = h.Auditor.RecordCalendar(r.Context(), !existed, user, uid)
		}
		w.WriteHeader(http.StatusCreated)
	case methodReport:
		h.report(w, r, user)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) propfind(w http.ResponseWriter, user string) {
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	_, _ = w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?>
<d:multistatus xmlns:d="DAV:" xmlns:cal="urn:ietf:params:xml:ns:caldav">
  <d:response>
    <d:href>/caldav/` + user + `/</d:href>
    <d:propstat>
      <d:prop><d:current-user-principal><d:href>/caldav/` + user + `/</d:href></d:current-user-principal></d:prop>
      <d:status>HTTP/1.1 200 OK</d:status>
    </d:propstat>
  </d:response>
</d:multistatus>`))
}

func (h *Handler) report(w http.ResponseWriter, _ *http.Request, user string) {
	events := h.Store.List(user)
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	b := strings.Builder{}
	b.WriteString(`<?xml version="1.0" encoding="utf-8"?><d:multistatus xmlns:d="DAV:" xmlns:cal="urn:ietf:params:xml:ns:caldav">`)
	for _, ev := range events {
		parsed, _ := ical.ParseVEVENT(ev.Body)
		b.WriteString(`<d:response><d:href>/caldav/` + user + `/` + ev.UID + `.ics</d:href>`)
		b.WriteString(`<d:propstat><d:prop><cal:calendar-data>`)
		b.WriteString(parsed.Summary)
		b.WriteString(`</cal:calendar-data></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat></d:response>`)
	}
	b.WriteString(`</d:multistatus>`)
	_, _ = w.Write([]byte(b.String()))
}

func (h *Handler) handleInviteReply(user string, parsed ical.Event) {
	att := parsed.Attendees[0]
	partstat := att.PartStat
	if partstat == "" {
		partstat = "ACCEPTED"
	}
	target, ok := h.Store.FindByUID(parsed.UID)
	if !ok {
		return
	}
	updated := ical.UpdateAttendeePartstat(target.Body, att.Email, partstat)
	h.Store.Put(target.Owner, target.UID, updated)
}

// WellKnown redirects /.well-known/caldav to user calendar home.
func WellKnown(st store.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.URL.Query().Get("user")
		if user == "" {
			user = "alice@mail.gov.az"
		}
		_ = st
		http.Redirect(w, r, "/caldav/"+user+"/", http.StatusPermanentRedirect)
	}
}
