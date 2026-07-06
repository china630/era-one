// STIX bundle ingest для internal TIP (S6-2).
package tip

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"time"
)

const stixBundleType = "bundle"
const stixIndicatorType = "indicator"
const stixSpecVersion = "2.1"

// STIXBundle — упрощённый STIX 2.1 (indicator-only, air-gap file load).
type STIXBundle struct {
	Type        string         `json:"type"`
	ID          string         `json:"id"`
	SpecVersion string         `json:"spec_version"`
	Objects     []STIXIndicator `json:"objects"`
}

type STIXIndicator struct {
	Type        string    `json:"type"`
	ID          string    `json:"id"`
	SpecVersion string    `json:"spec_version"`
	Name        string    `json:"name"`
	Pattern     string    `json:"pattern"`
	PatternType string    `json:"pattern_type"`
	ValidFrom   time.Time `json:"valid_from"`
	Confidence  int       `json:"confidence"`
	Labels      []string  `json:"labels,omitempty"`
}

// LoadSTIXBundle читает STIX bundle с диска и возвращает Feed с извлечёнными IOC.
func LoadSTIXBundle(path string) (*Feed, error) {
	if path == "" {
		return &Feed{}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return FeedFromSTIX(data)
}

// FeedFromSTIX парсит STIX JSON и нормализует patterns в строки для Match.
func FeedFromSTIX(data []byte) (*Feed, error) {
	var b STIXBundle
	if err := json.Unmarshal(data, &b); err != nil {
		return nil, err
	}
	if b.Type != "" && b.Type != stixBundleType {
		return nil, errors.New("tip: not a STIX bundle")
	}
	var patterns []string
	for _, obj := range b.Objects {
		if obj.Type != "" && obj.Type != stixIndicatorType {
			continue
		}
		for _, ioc := range extractIOCs(obj.Pattern) {
			if ioc != "" {
				patterns = append(patterns, ioc)
			}
		}
	}
	return FromPatterns(patterns), nil
}

func extractIOCs(pattern string) []string {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return nil
	}
	// STIX pattern: [domain-name:value = 'evil.tld'] → evil.tld
	if i := strings.Index(pattern, "'"); i >= 0 {
		if j := strings.LastIndex(pattern, "'"); j > i {
			return []string{pattern[i+1 : j]}
		}
	}
	return []string{pattern}
}
