package store

import "time"

// HybridOps — persistence hybrid policy/runtime/audit (ADR-0018).
type HybridOps interface {
	GetHybridPolicy() HybridPolicy
	SetHybridPolicy(HybridPolicy)
	GetHybridRuntime() HybridRuntime
	SetHybridRuntime(HybridRuntime)
	RecordEgressAudit(e *EgressAuditEntry)
	ListEgressAudit(limit int) []*EgressAuditEntry
	GetLeaseCache() (token string, lastRenew time.Time)
	SetLeaseCache(token string, lastRenew time.Time)
}
