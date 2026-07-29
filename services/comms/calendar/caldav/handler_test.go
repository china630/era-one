package caldav

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"era/services/comms/calendar/store"
)

func TestCalDAVPutGetReport(t *testing.T) {
	st := store.New()
	h := &Handler{Store: st}
	body := `BEGIN:VCALENDAR
BEGIN:VEVENT
UID:evt-1
SUMMARY:Team Standup
DTSTART:20260707T090000Z
END:VEVENT
END:VCALENDAR`

	req := httptest.NewRequest(http.MethodPut, "/caldav/alice@mail.gov.az/evt-1.ics", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("put %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/caldav/alice@mail.gov.az/evt-1.ics", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get %d", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("Team Standup")) {
		t.Fatalf("body %s", rec.Body.String())
	}

	req = httptest.NewRequest(methodReport, "/caldav/alice@mail.gov.az/", strings.NewReader("<calendar-query/>"))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("report %d", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("Team Standup")) {
		t.Fatalf("report body %s", rec.Body.String())
	}
}

func TestCalDAVPropfind(t *testing.T) {
	h := &Handler{Store: store.New()}
	req := httptest.NewRequest(methodPropfind, "/caldav/alice@mail.gov.az/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("propfind %d", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("current-user-principal")) {
		t.Fatalf("body %s", rec.Body.String())
	}
}

func TestCalDAVUpdateEvent(t *testing.T) {
	st := store.New()
	h := &Handler{Store: st}
	user := "alice@mail.gov.az"
	uid := "evt-1"
	body1 := `BEGIN:VEVENT
UID:evt-1
SUMMARY:Standup
DTSTART:20260707T090000Z
END:VEVENT`
	req := httptest.NewRequest(http.MethodPut, "/caldav/"+user+"/"+uid+".ics", strings.NewReader(body1))
	h.ServeHTTP(httptest.NewRecorder(), req)

	body2 := `BEGIN:VEVENT
UID:evt-1
SUMMARY:Standup
DTSTART:20260707T100000Z
END:VEVENT`
	req = httptest.NewRequest(http.MethodPut, "/caldav/"+user+"/"+uid+".ics", strings.NewReader(body2))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("update put %d", rec.Code)
	}
	ev, ok := st.Get(user, uid)
	if !ok || !strings.Contains(ev.Body, "100000Z") {
		t.Fatalf("updated body %s", ev.Body)
	}
}

func TestCalDAVInviteReplyGolden(t *testing.T) {
	want, err := os.ReadFile("testdata/invite_reply.golden.ics")
	if err != nil {
		t.Fatal(err)
	}
	st := store.New()
	h := &Handler{Store: st}
	organizer := "alice@mail.gov.az"
	attendee := "bob@mail.gov.az"
	uid := "invite-standup-001"
	invite := `BEGIN:VCALENDAR
VERSION:2.0
METHOD:REQUEST
BEGIN:VEVENT
UID:` + uid + `
ORGANIZER:mailto:` + organizer + `
ATTENDEE;PARTSTAT=NEEDS-ACTION;RSVP=TRUE:mailto:` + attendee + `
SUMMARY:Gov Standup
DTSTART:20260707T090000Z
END:VEVENT
END:VCALENDAR`
	req := httptest.NewRequest(http.MethodPut, "/caldav/"+organizer+"/"+uid+".ics", strings.NewReader(invite))
	h.ServeHTTP(httptest.NewRecorder(), req)

	reply := `BEGIN:VCALENDAR
VERSION:2.0
METHOD:REPLY
BEGIN:VEVENT
UID:` + uid + `
ATTENDEE;PARTSTAT=ACCEPTED:mailto:` + attendee + `
END:VEVENT
END:VCALENDAR`
	req = httptest.NewRequest(http.MethodPut, "/caldav/"+attendee+"/"+uid+".ics", strings.NewReader(reply))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("reply status %d", rec.Code)
	}
	ev, ok := st.Get(organizer, uid)
	if !ok {
		t.Fatal("organizer event missing")
	}
	if !strings.Contains(ev.Body, "PARTSTAT=ACCEPTED") {
		t.Fatalf("partstat not updated: %s", ev.Body)
	}
	if !bytes.Contains([]byte(ev.Body), []byte("bob@mail.gov.az")) {
		t.Fatalf("attendee missing: %s", ev.Body)
	}
	if !bytes.EqualFold(normalizeICal(ev.Body), normalizeICal(string(want))) {
		t.Fatalf("golden mismatch\ngot:\n%s\nwant:\n%s", ev.Body, string(want))
	}
}

func normalizeICal(s string) []byte {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	return []byte(strings.Join(out, "\n"))
}
