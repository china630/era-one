package policy

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

type Action string

const (
	ActionAllow Action = "allow"
	ActionDeny  Action = "deny"
)

type Rule struct {
	ID       string `json:"id"`
	Action   Action `json:"action"`
	SrcCIDR  string `json:"src_cidr,omitempty"`
	DstCIDR  string `json:"dst_cidr,omitempty"`
	DstPort  uint32 `json:"dst_port,omitempty"`
	Protocol string `json:"protocol,omitempty"`
}

type Decision struct {
	Allowed bool   `json:"allowed"`
	RuleID  string `json:"rule_id"`
	Reason  string `json:"reason"`
}

type Engine struct {
	mu   sync.RWMutex
	Rules []Rule
	path string
}

func Default() *Engine {
	return &Engine{Rules: []Rule{
		{ID: "allow-internal", Action: ActionAllow, DstCIDR: "10.0.0.0/8"},
		{ID: "allow-internal-192", Action: ActionAllow, DstCIDR: "192.168.0.0/16"},
		{ID: "deny-external-smb", Action: ActionDeny, DstPort: 445, SrcCIDR: "0.0.0.0/0"},
		{ID: "deny-external-rdp", Action: ActionDeny, DstPort: 3389, SrcCIDR: "0.0.0.0/0"},
	}}
}

func (e *Engine) Evaluate(srcIP, dstIP, protocol string, dstPort uint32) Decision {
	e.mu.RLock()
	defer e.mu.RUnlock()
	for _, pass := range []Action{ActionDeny, ActionAllow} {
		for _, rule := range e.Rules {
			if rule.Action != pass {
				continue
			}
			if !ruleMatches(rule, srcIP, dstIP, protocol, dstPort) {
				continue
			}
			switch rule.Action {
			case ActionDeny:
				return Decision{Allowed: false, RuleID: rule.ID, Reason: "policy deny"}
			case ActionAllow:
				return Decision{Allowed: true, RuleID: rule.ID, Reason: "policy allow"}
			}
		}
	}
	return Decision{Allowed: true, RuleID: "default-allow", Reason: "implicit allow"}
}

func (e *Engine) List() []Rule {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]Rule, len(e.Rules))
	copy(out, e.Rules)
	return out
}

func (e *Engine) Get(id string) (Rule, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	for _, r := range e.Rules {
		if r.ID == id {
			return r, true
		}
	}
	return Rule{}, false
}

// GetByIndex returns a rule by zero-based index.
func (e *Engine) GetByIndex(i int) (Rule, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if i < 0 || i >= len(e.Rules) {
		return Rule{}, false
	}
	return e.Rules[i], true
}

func (e *Engine) Replace(rules []Rule) error {
	e.mu.Lock()
	e.Rules = append([]Rule(nil), rules...)
	path := e.path
	e.mu.Unlock()
	if path != "" {
		return e.Save(path)
	}
	return nil
}

func (e *Engine) Save(path string) error {
	e.mu.RLock()
	defer e.mu.RUnlock()
	data, err := json.MarshalIndent(e.Rules, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func (e *Engine) Load(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var rules []Rule
	if err := json.Unmarshal(data, &rules); err != nil {
		return err
	}
	e.mu.Lock()
	e.Rules = rules
	e.path = path
	e.mu.Unlock()
	return nil
}

func (e *Engine) SetPath(path string) { e.path = path }

func ruleMatches(rule Rule, srcIP, dstIP, protocol string, dstPort uint32) bool {
	if rule.Protocol != "" && !strings.EqualFold(rule.Protocol, protocol) {
		return false
	}
	if rule.DstPort != 0 && rule.DstPort != dstPort {
		return false
	}
	if rule.SrcCIDR != "" && !inCIDR(srcIP, rule.SrcCIDR) {
		return false
	}
	if rule.DstCIDR != "" && !inCIDR(dstIP, rule.DstCIDR) {
		return false
	}
	return true
}

func inCIDR(ipStr, cidr string) bool {
	if cidr == "0.0.0.0/0" {
		return true
	}
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	_, n, err := net.ParseCIDR(cidr)
	if err != nil {
		return false
	}
	return n.Contains(ip)
}

type Flow struct {
	SrcIP    string `json:"src_ip"`
	DstIP    string `json:"dst_ip"`
	Protocol string `json:"protocol"`
	DstPort  uint32 `json:"dst_port"`
}

func (f Flow) String() string {
	return fmt.Sprintf("%s -> %s:%d/%s", f.SrcIP, f.DstIP, f.DstPort, f.Protocol)
}
