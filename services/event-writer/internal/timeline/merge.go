// Package timeline — merge событий и детекций для Workbench (ADR-0017 §1).
package timeline

import (
	"sort"
	"time"
)

// Entry — элемент единого incident timeline.
type Entry struct {
	Kind          string `json:"kind"` // event | detection
	At            string `json:"at"`
	NodeID        string `json:"node_id"`
	Severity      string `json:"severity"`
	Category      string `json:"category,omitempty"`
	Source        string `json:"source,omitempty"`
	Summary       string `json:"summary"`
	EventID       string `json:"event_id,omitempty"`
	DetectionID   string `json:"detection_id,omitempty"`
	CorrelationID string `json:"correlation_id,omitempty"`
	Payload       string `json:"payload,omitempty"`
}

// Merge сортирует events+detections по времени (asc — хронология расследования).
func Merge(events, detections []Entry) []Entry {
	out := make([]Entry, 0, len(events)+len(detections))
	out = append(out, events...)
	out = append(out, detections...)
	sort.Slice(out, func(i, j int) bool {
		ti, _ := time.Parse(time.RFC3339Nano, out[i].At)
		tj, _ := time.Parse(time.RFC3339Nano, out[j].At)
		if ti.Equal(tj) {
			return out[i].Kind < out[j].Kind
		}
		return ti.Before(tj)
	})
	return out
}
