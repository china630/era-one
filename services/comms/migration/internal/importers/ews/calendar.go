package ews

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"
)

// CalendarItem — migrated calendar event.
type CalendarItem struct {
	ID      string
	Subject string
	Start   time.Time
	End     time.Time
	ICS     string
}

// ImportCalendar counts imported items (legacy smoke).
func ImportCalendar(items []CalendarItem) int {
	return len(items)
}

// ParseICS extracts VEVENT subset from iCalendar text.
func ParseICS(raw string) ([]CalendarItem, error) {
	var items []CalendarItem
	var cur CalendarItem
	inEvent := false
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case line == "BEGIN:VEVENT":
			inEvent = true
			cur = CalendarItem{}
		case line == "END:VEVENT" && inEvent:
			inEvent = false
			if cur.ID == "" {
				cur.ID = fmt.Sprintf("evt-%d", len(items)+1)
			}
			cur.ICS = buildICS(cur)
			items = append(items, cur)
		case inEvent && strings.HasPrefix(line, "UID:"):
			cur.ID = strings.TrimPrefix(line, "UID:")
		case inEvent && strings.HasPrefix(line, "SUMMARY:"):
			cur.Subject = strings.TrimPrefix(line, "SUMMARY:")
		case inEvent && strings.HasPrefix(line, "DTSTART"):
			cur.Start = parseICSTime(line)
		case inEvent && strings.HasPrefix(line, "DTEND"):
			cur.End = parseICSTime(line)
		}
	}
	return items, nil
}

// ImportICSFile reads .ics and returns calendar items for target writer.
func ImportICSFile(path string) ([]CalendarItem, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseICS(string(b))
}

func buildICS(item CalendarItem) string {
	var buf bytes.Buffer
	buf.WriteString("BEGIN:VCALENDAR\r\nBEGIN:VEVENT\r\n")
	buf.WriteString("UID:" + item.ID + "\r\n")
	buf.WriteString("SUMMARY:" + item.Subject + "\r\n")
	if !item.Start.IsZero() {
		buf.WriteString("DTSTART:" + item.Start.UTC().Format("20060102T150405Z") + "\r\n")
	}
	if !item.End.IsZero() {
		buf.WriteString("DTEND:" + item.End.UTC().Format("20060102T150405Z") + "\r\n")
	}
	buf.WriteString("END:VEVENT\r\nEND:VCALENDAR\r\n")
	return buf.String()
}

func parseICSTime(line string) time.Time {
	if i := strings.Index(line, ":"); i >= 0 {
		line = line[i+1:]
	}
	t, _ := time.Parse("20060102T150405Z", strings.TrimSpace(line))
	return t
}
