package policy

import (
	"encoding/json"
	"os"
	"strings"
	"sync"
)

// Action is a Guard decision.
type Action string

const (
	ActionAllow    Action = "allow"
	ActionNXDomain Action = "nxdomain"
	ActionSinkhole Action = "sinkhole"
)

// Rule matches a domain or suffix.
type Rule struct {
	ID         string `json:"id"`
	Domain     string `json:"domain,omitempty"`  // exact (case-insensitive)
	Suffix     string `json:"suffix,omitempty"`  // e.g. .evil.test
	Action     Action `json:"action"`
	SinkholeIP string `json:"sinkhole_ip,omitempty"`
	Reason     string `json:"reason,omitempty"`
}

// Store holds ordered policy rules (first match wins).
type Store struct {
	mu    sync.RWMutex
	Rules []Rule
}

func NewStore() *Store {
	return &Store{Rules: []Rule{
		{ID: "block-malware-lab", Suffix: ".malware.test", Action: ActionNXDomain, Reason: "lab deny"},
		{ID: "sinkhole-phish", Suffix: ".phish.test", Action: ActionSinkhole, SinkholeIP: "127.0.0.1", Reason: "lab sinkhole"},
	}}
}

func (s *Store) List() []Rule {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Rule, len(s.Rules))
	copy(out, s.Rules)
	return out
}

func (s *Store) Replace(rules []Rule) {
	s.mu.Lock()
	s.Rules = append([]Rule(nil), rules...)
	s.mu.Unlock()
}

// Add appends one rule (or replaces existing id).
func (s *Store) Add(rule Rule) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if rule.ID != "" {
		for i, r := range s.Rules {
			if r.ID == rule.ID {
				s.Rules[i] = rule
				return
			}
		}
	}
	s.Rules = append(s.Rules, rule)
}

// Delete removes a rule by id. Returns false if not found.
func (s *Store) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, r := range s.Rules {
		if r.ID == id {
			s.Rules = append(s.Rules[:i], s.Rules[i+1:]...)
			return true
		}
	}
	return false
}

// Get returns a rule by id.
func (s *Store) Get(id string) (Rule, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, r := range s.Rules {
		if r.ID == id {
			return r, true
		}
	}
	return Rule{}, false
}

func (s *Store) LoadFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var rules []Rule
	if err := json.Unmarshal(data, &rules); err != nil {
		return err
	}
	s.Replace(rules)
	return nil
}

// Match returns the first matching rule for qname.
func (s *Store) Match(qname string) (Rule, bool) {
	name := normalize(qname)
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, r := range s.Rules {
		if r.Domain != "" && normalize(r.Domain) == name {
			return r, true
		}
		if r.Suffix != "" {
			suf := normalize(r.Suffix)
			if !strings.HasPrefix(suf, ".") {
				suf = "." + suf
			}
			if strings.HasSuffix(name, suf) || name == strings.TrimPrefix(suf, ".") {
				return r, true
			}
		}
	}
	return Rule{}, false
}

func normalize(s string) string {
	s = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(s)), ".")
	return s
}
