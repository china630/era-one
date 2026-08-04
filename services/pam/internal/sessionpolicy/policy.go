package sessionpolicy

import (
	"os"
	"strconv"
	"sync"
	"time"
)

// Policy holds idle/max duration for privileged sessions.
type Policy struct {
	MaxDuration time.Duration
	IdleTimeout time.Duration
}

func FromEnv() Policy {
	return Policy{
		MaxDuration: durEnv("ERA_PAM_SESSION_MAX", 8*time.Hour),
		IdleTimeout: durEnv("ERA_PAM_SESSION_IDLE", 30*time.Minute),
	}
}

func durEnv(k string, def time.Duration) time.Duration {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	if sec, err := strconv.Atoi(v); err == nil && sec > 0 {
		return time.Duration(sec) * time.Second
	}
	if d, err := time.ParseDuration(v); err == nil {
		return d
	}
	return def
}

// Tracker watches sessions for timeout.
type Tracker struct {
	mu       sync.Mutex
	started  map[string]time.Time
	lastAct  map[string]time.Time
	policy   Policy
	onExpire func(sessionID, reason string)
}

func NewTracker(p Policy, onExpire func(string, string)) *Tracker {
	return &Tracker{
		started: map[string]time.Time{}, lastAct: map[string]time.Time{},
		policy: p, onExpire: onExpire,
	}
}

func (t *Tracker) Start(sessionID string) {
	now := time.Now().UTC()
	t.mu.Lock()
	t.started[sessionID] = now
	t.lastAct[sessionID] = now
	t.mu.Unlock()
}

func (t *Tracker) Touch(sessionID string) {
	t.mu.Lock()
	t.lastAct[sessionID] = time.Now().UTC()
	t.mu.Unlock()
}

func (t *Tracker) Stop(sessionID string) {
	t.mu.Lock()
	delete(t.started, sessionID)
	delete(t.lastAct, sessionID)
	t.mu.Unlock()
}

// Sweep expires sessions; call periodically or from tests.
func (t *Tracker) Sweep(now time.Time) []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	var expired []string
	for id, start := range t.started {
		reason := ""
		if t.policy.MaxDuration > 0 && now.Sub(start) >= t.policy.MaxDuration {
			reason = "max_duration"
		} else if t.policy.IdleTimeout > 0 && now.Sub(t.lastAct[id]) >= t.policy.IdleTimeout {
			reason = "idle_timeout"
		}
		if reason != "" {
			expired = append(expired, id)
			if t.onExpire != nil {
				t.onExpire(id, reason)
			}
			delete(t.started, id)
			delete(t.lastAct, id)
		}
	}
	return expired
}
