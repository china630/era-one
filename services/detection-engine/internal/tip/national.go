// Package tip — national IOC feed for detection boost (F4-3).
package tip

import (
	"encoding/json"
	"os"
	"strings"
)

type Feed struct {
	patterns []string
}

func FromPatterns(patterns []string) *Feed {
	return &Feed{patterns: patterns}
}

func LoadFile(path string) (*Feed, error) {
	if path == "" {
		return &Feed{}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var patterns []string
	if err := json.Unmarshal(data, &patterns); err != nil {
		return nil, err
	}
	return &Feed{patterns: patterns}, nil
}

func (f *Feed) Match(payload string) (bool, string) {
	if f == nil {
		return false, ""
	}
	low := strings.ToLower(payload)
	for _, p := range f.patterns {
		if p != "" && strings.Contains(low, strings.ToLower(p)) {
			return true, "era-national-ioc"
		}
	}
	return false, ""
}

func (f *Feed) PatternCount() int {
	if f == nil {
		return 0
	}
	return len(f.patterns)
}
