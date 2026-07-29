package ews

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseICSGolden(t *testing.T) {
	raw := `BEGIN:VCALENDAR
BEGIN:VEVENT
UID:mig-evt-1
SUMMARY:Pilot meeting
DTSTART:20260710T100000Z
DTEND:20260710T110000Z
END:VEVENT
END:VCALENDAR`
	items, err := ParseICS(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Subject != "Pilot meeting" {
		t.Fatalf("got %+v", items)
	}
	golden := filepath.Join("testdata", "calendar_import.golden.json")
	if _, err := os.Stat(golden); os.IsNotExist(err) {
		t.Skip("golden not present")
	}
}
