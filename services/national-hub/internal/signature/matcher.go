// Package signature — национальные IOC-сигнатуры для подписчиков (F4-3).
package signature

import (
	"strings"

	"era/services/national-hub/internal/stix"
)

type Rule struct {
	ID      string
	Pattern string
	Source  string
}

type Matcher struct {
	rules []Rule
}

func FromBundle(b *stix.Bundle, source string) *Matcher {
	var rules []Rule
	for _, obj := range b.Objects {
		pat := normalizePattern(obj.Pattern)
		if pat == "" {
			continue
		}
		rules = append(rules, Rule{ID: obj.ID, Pattern: pat, Source: source})
	}
	return &Matcher{rules: rules}
}

func (m *Matcher) Rules() []Rule {
	if m == nil {
		return nil
	}
	return m.rules
}

func NewMatcher(rules []Rule) *Matcher {
	return &Matcher{rules: rules}
}

func (m *Matcher) Match(payload string) int {
	if m == nil {
		return 0
	}
	low := strings.ToLower(payload)
	n := 0
	for _, r := range m.rules {
		if strings.Contains(low, strings.ToLower(r.Pattern)) {
			n++
		}
	}
	return n
}

func normalizePattern(p string) string {
	// [domain-name:value='evil.az'] -> evil.az
	if i := strings.Index(p, "value='"); i >= 0 {
		rest := p[i+7:]
		if j := strings.Index(rest, "'"); j >= 0 {
			return rest[:j]
		}
	}
	if i := strings.Index(p, "value=\""); i >= 0 {
		rest := p[i+7:]
		if j := strings.Index(rest, "\""); j >= 0 {
			return rest[:j]
		}
	}
	return p
}

// DetectionDelta returns additional matches with national sigs vs baseline.
func DetectionDelta(baseline, withNational int) float64 {
	if baseline == 0 && withNational > 0 {
		return 1.0
	}
	if baseline == 0 {
		return 0
	}
	return float64(withNational-baseline) / float64(baseline)
}
