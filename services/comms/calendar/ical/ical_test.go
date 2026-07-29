package ical

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSerializeParseGolden(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "event_standup.ics"))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseVEVENT(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	if parsed.UID != "standup-uid-001" {
		t.Fatalf("uid %q", parsed.UID)
	}
	if parsed.Summary != "Team Standup" {
		t.Fatalf("summary %q", parsed.Summary)
	}
	out := SerializeVEVENT(parsed)
	parsed2, err := ParseVEVENT(out)
	if err != nil {
		t.Fatal(err)
	}
	if parsed2.UID != parsed.UID || parsed2.Summary != parsed.Summary {
		t.Fatalf("roundtrip mismatch: %+v vs %+v", parsed2, parsed)
	}
}

func TestUpdateDTSTART(t *testing.T) {
	raw := "BEGIN:VEVENT\nUID:x\nDTSTART:20260707T090000Z\nEND:VEVENT"
	updated := UpdateDTSTART(raw, "20260707T100000Z")
	if !stringsContains(updated, "DTSTART:20260707T100000Z") {
		t.Fatalf("dtstart not updated: %s", updated)
	}
}

func stringsContains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
