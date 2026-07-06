// Package stix — упрощённый STIX 2.1 (indicator-only, без PII, Фаза 4).
package stix

import (
	"encoding/json"
	"time"
)

const BundleType = "bundle"
const IndicatorType = "indicator"
const SpecVersion = "2.1"

type Bundle struct {
	Type        string      `json:"type"`
	ID          string      `json:"id"`
	SpecVersion string      `json:"spec_version"`
	Objects     []Indicator `json:"objects"`
}

type Indicator struct {
	Type        string    `json:"type"`
	ID          string    `json:"id"`
	SpecVersion string    `json:"spec_version"`
	Created     time.Time `json:"created"`
	Modified    time.Time `json:"modified"`
	Name        string    `json:"name"`
	Pattern     string    `json:"pattern"`
	PatternType string    `json:"pattern_type"`
	ValidFrom   time.Time `json:"valid_from"`
	Confidence  int       `json:"confidence"`
	Labels      []string  `json:"labels,omitempty"`
}

func ParseBundle(data []byte) (*Bundle, error) {
	var b Bundle
	if err := json.Unmarshal(data, &b); err != nil {
		return nil, err
	}
	return &b, nil
}

func (b *Bundle) JSON() ([]byte, error) {
	return json.Marshal(b)
}

// ExtractIOCs returns normalized IOC strings from STIX patterns.
func (b *Bundle) ExtractIOCs() []string {
	var out []string
	for _, obj := range b.Objects {
		if obj.Pattern != "" {
			out = append(out, obj.Pattern)
		}
	}
	return out
}
