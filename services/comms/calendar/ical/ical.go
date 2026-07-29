// Package ical — minimal VEVENT parse/serialize (golden-tested).
package ical

import (
	"fmt"
	"strings"
	"time"
)

// Attendee is a parsed ATTENDEE line (iTIP subset).
type Attendee struct {
	Email    string
	PartStat string
}

// Event is a parsed VEVENT subset.
type Event struct {
	UID        string
	Summary    string
	DTStart    time.Time
	DTStartRaw string
	Method     string
	Organizer  string
	Attendees  []Attendee
}

// ParseVEVENT extracts fields from iCal text.
func ParseVEVENT(raw string) (Event, error) {
	var ev Event
	lines := unfoldLines(raw)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "METHOD:") {
			ev.Method = strings.TrimPrefix(line, "METHOD:")
			continue
		}
		if strings.HasPrefix(line, "UID:") {
			ev.UID = strings.TrimPrefix(line, "UID:")
			continue
		}
		if strings.HasPrefix(line, "SUMMARY:") {
			ev.Summary = strings.TrimPrefix(line, "SUMMARY:")
			continue
		}
		if strings.HasPrefix(line, "ORGANIZER") {
			ev.Organizer = parseMailto(line)
			continue
		}
		if strings.HasPrefix(line, "ATTENDEE") {
			if att := parseAttendee(line); att.Email != "" {
				ev.Attendees = append(ev.Attendees, att)
			}
			continue
		}
		if strings.HasPrefix(line, "DTSTART") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				ev.DTStartRaw = parts[1]
				if t, err := time.Parse("20060102T150405Z", parts[1]); err == nil {
					ev.DTStart = t.UTC()
				}
			}
		}
	}
	if ev.UID == "" {
		return ev, fmt.Errorf("ical: missing UID")
	}
	return ev, nil
}

func unfoldLines(raw string) []string {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	lines := strings.Split(raw, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if len(out) > 0 && (strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")) {
			out[len(out)-1] += strings.TrimLeft(line, " \t")
			continue
		}
		out = append(out, line)
	}
	return out
}

func parseMailto(line string) string {
	if idx := strings.Index(line, "mailto:"); idx >= 0 {
		addr := line[idx+len("mailto:"):]
		if semi := strings.Index(addr, ";"); semi >= 0 {
			addr = addr[:semi]
		}
		return strings.TrimSpace(addr)
	}
	parts := strings.SplitN(line, ":", 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[1])
	}
	return ""
}

func parseAttendee(line string) Attendee {
	var att Attendee
	upper := strings.ToUpper(line)
	if i := strings.Index(upper, "PARTSTAT="); i >= 0 {
		rest := line[i+len("PARTSTAT="):]
		if semi := strings.Index(rest, ";"); semi >= 0 {
			att.PartStat = strings.TrimSpace(rest[:semi])
		} else if colon := strings.Index(rest, ":"); colon >= 0 {
			att.PartStat = strings.TrimSpace(rest[:colon])
		} else {
			att.PartStat = strings.TrimSpace(rest)
		}
	}
	att.Email = parseMailto(line)
	return att
}

// UpdateAttendeePartstat rewrites PARTSTAT for the matching attendee email.
func UpdateAttendeePartstat(raw, attendeeEmail, partstat string) string {
	attendeeEmail = strings.ToLower(strings.TrimSpace(attendeeEmail))
	partstat = strings.ToUpper(strings.TrimSpace(partstat))
	lines := unfoldLines(raw)
	for i, line := range lines {
		if !strings.HasPrefix(strings.ToUpper(line), "ATTENDEE") {
			continue
		}
		if strings.ToLower(parseMailto(line)) != attendeeEmail {
			continue
		}
		upper := strings.ToUpper(line)
		if strings.Contains(upper, "PARTSTAT=") {
			start := strings.Index(upper, "PARTSTAT=")
			rest := line[start+len("PARTSTAT="):]
			end := len(rest)
			for j, ch := range rest {
				if ch == ';' || ch == ':' {
					end = j
					break
				}
			}
			lines[i] = line[:start+len("PARTSTAT=")] + partstat + rest[end:]
		} else {
			colon := strings.Index(line, ":")
			if colon >= 0 {
				lines[i] = line[:colon] + ";PARTSTAT=" + partstat + line[colon:]
			}
		}
	}
	return strings.Join(lines, "\r\n")
}

// SerializeVEVENT builds canonical iCal for storage.
func SerializeVEVENT(ev Event) string {
	dt := ev.DTStartRaw
	if dt == "" && !ev.DTStart.IsZero() {
		dt = ev.DTStart.UTC().Format("20060102T150405Z")
	}
	sum := ev.Summary
	if sum == "" {
		sum = "Event"
	}
	return fmt.Sprintf("BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//ERA Communications//CalDAV//EN\r\nBEGIN:VEVENT\r\nUID:%s\r\nSUMMARY:%s\r\nDTSTART:%s\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n",
		ev.UID, sum, dt)
}

// UpdateDTSTART replaces DTSTART in raw iCal body.
func UpdateDTSTART(raw, newDT string) string {
	lines := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "DTSTART") {
			if idx := strings.Index(line, ":"); idx >= 0 {
				lines[i] = line[:idx+1] + newDT
			}
		}
	}
	return strings.Join(lines, "\r\n")
}
