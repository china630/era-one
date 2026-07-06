// Package correlator — кросс-доменная корреляция (F2-3 APT-сценарий).
package correlator

import (
	"strings"
	"sync"
	"time"

	erav1 "era/contracts/gen/era/v1"
)

type EventRecord struct {
	NodeID    string
	Category  string
	EventID   string
	Observed  time.Time
	Payload   string
}

type Engine struct {
	mu     sync.Mutex
	window time.Duration
	byNode map[string][]EventRecord
}

func New(window time.Duration) *Engine {
	return &Engine{
		window: window,
		byNode: make(map[string][]EventRecord),
	}
}

func (e *Engine) Observe(nodeID, category, eventID, payload string, at time.Time) {
	e.mu.Lock()
	defer e.mu.Unlock()
	cutoff := at.Add(-e.window)
	list := e.byNode[nodeID]
	var kept []EventRecord
	for _, r := range list {
		if r.Observed.After(cutoff) {
			kept = append(kept, r)
		}
	}
	kept = append(kept, EventRecord{
		NodeID: nodeID, Category: category, EventID: eventID, Observed: at, Payload: payload,
	})
	e.byNode[nodeID] = kept
}

// APTChain: process (suspicious) → network (internal) → auth (failed).
func (e *Engine) APTChain(nodeID string) (bool, string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	list := e.byNode[nodeID]
	var hasProcess, hasNetwork, hasAuth bool
	for _, r := range list {
		switch r.Category {
		case "process":
			if containsAny(r.Payload, "powershell", "cmd.exe", "wscript") {
				hasProcess = true
			}
		case "network":
			if containsAny(r.Payload, "10.", "192.168.", "172.16.") {
				hasNetwork = true
			}
		case "auth":
			if containsAny(r.Payload, "failed", "false") {
				hasAuth = true
			}
		}
	}
	if hasProcess && hasNetwork && hasAuth {
		return true, "era-apt-lateral-movement"
	}
	return false, ""
}

// ObserveNetworkEndpoint: NMS high-egress alert + suspicious process on same node.
func (e *Engine) ObserveNetworkEndpoint(nodeID string) (bool, string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	list := e.byNode[nodeID]
	var hasNMS, hasSuspiciousProcess bool
	for _, r := range list {
		switch r.Category {
		case "network":
			if containsAny(r.Payload, "high egress", "prtg", "zabbix", "observe_snmp", "netflow") {
				hasNMS = true
			}
		case "process":
			if containsAny(r.Payload, "suspicious", "malware", "powershell", "cmd.exe") {
				hasSuspiciousProcess = true
			}
		}
	}
	if hasNMS && hasSuspiciousProcess {
		return true, "era-observe-network-endpoint"
	}
	return false, ""
}

// CategoryName maps proto category to sigma logsource category.
func CategoryName(c erav1.EventCategory) string {
	switch c {
	case erav1.EventCategory_EVENT_CATEGORY_PROCESS:
		return "process"
	case erav1.EventCategory_EVENT_CATEGORY_NETWORK:
		return "network"
	case erav1.EventCategory_EVENT_CATEGORY_AUTH:
		return "auth"
	case erav1.EventCategory_EVENT_CATEGORY_FILE:
		return "file"
	default:
		return "other"
	}
}

func containsAny(s string, subs ...string) bool {
	low := strings.ToLower(s)
	for _, sub := range subs {
		if strings.Contains(low, strings.ToLower(sub)) {
			return true
		}
	}
	return false
}
