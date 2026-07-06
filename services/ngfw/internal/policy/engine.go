// Package policy — NGFW/eBPF-подобный движок сетевых политик (Cilium-паттерн, F3-2).
package policy

import (
	"fmt"
	"net"
	"strings"
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
	Allowed bool
	RuleID  string
	Reason  string
}

type Engine struct {
	Rules []Rule
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
