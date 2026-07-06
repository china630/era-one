// Package honey — deception / honeypot (ADR-0006 P1, Фаза 3).
package honey

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type Touch struct {
	ID        string    `json:"id"`
	Path      string    `json:"path"`
	RemoteIP  string    `json:"remote_ip"`
	UserAgent string    `json:"user_agent"`
	At        time.Time `json:"at"`
	Honeytoken bool     `json:"honeytoken,omitempty"`
}

// DetectionEvent — срабатывание deception touch rule.
type DetectionEvent struct {
	RuleID   string `json:"rule_id"`
	Title    string `json:"title"`
	Severity string `json:"severity"`
	Touch    Touch  `json:"touch"`
}

// Honeytoken — фиктивные учётные данные (canary).
type Honeytoken struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Path     string `json:"path"`
}

type Engine struct {
	mu         sync.Mutex
	touches    []Touch
	decoys     map[string]bool
	honeytokens map[string]Honeytoken
}

func New() *Engine {
	tokens := map[string]Honeytoken{
		"/decoy/creds/.env": {ID: "ht-001", Username: "svc_backup", Path: "/decoy/creds/.env"},
		"/decoy/creds/db.json": {ID: "ht-002", Username: "admin_ro", Path: "/decoy/creds/db.json"},
	}
	decoys := map[string]bool{
		"/decoy/admin":        true,
		"/decoy/.env":         true,
		"/decoy/backup.sql":   true,
		"/decoy/creds/.env":   true,
		"/decoy/creds/db.json": true,
	}
	return &Engine{
		decoys:      decoys,
		honeytokens: tokens,
	}
}

func (e *Engine) IsDecoy(path string) bool {
	return e.decoys[path]
}

func (e *Engine) Honeytokens() []Honeytoken {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]Honeytoken, 0, len(e.honeytokens))
	for _, t := range e.honeytokens {
		out = append(out, t)
	}
	return out
}

func (e *Engine) Record(path, remoteIP, ua string) Touch {
	e.mu.Lock()
	defer e.mu.Unlock()
	_, isHT := e.honeytokens[path]
	t := Touch{
		ID: uuid.NewString(), Path: path, RemoteIP: remoteIP,
		UserAgent: ua, At: time.Now().UTC(), Honeytoken: isHT,
	}
	e.touches = append(e.touches, t)
	return t
}

// MatchTouchRule возвращает detection при touch decoy/honeytoken.
func (e *Engine) MatchTouchRule(t Touch) (bool, DetectionEvent) {
	if !e.decoys[t.Path] {
		return false, DetectionEvent{}
	}
	ruleID := "era-deception-decoy-touch"
	title := "Deception decoy accessed"
	severity := "high"
	if t.Honeytoken {
		ruleID = "era-deception-honeytoken-touch"
		title = "Honeytoken credentials accessed"
		severity = "critical"
	}
	return true, DetectionEvent{
		RuleID: ruleID, Title: title, Severity: severity, Touch: t,
	}
}

func (e *Engine) Touches() []Touch {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]Touch, len(e.touches))
	copy(out, e.touches)
	return out
}
