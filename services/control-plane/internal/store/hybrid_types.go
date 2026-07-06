package store

import "time"

// HybridPolicy — tenant policy для connected-режима (ADR-0018 §2.2, §4).
type HybridPolicy struct {
	Enabled         bool     `json:"enabled"`
	PortalURL       string   `json:"portal_url"`
	UpdateURL       string   `json:"update_url"`
	EgressAllowlist []string `json:"egress_allowlist"`
	HealthLevel     string   `json:"health_level"` // A | B | C
	TIShare         bool     `json:"ti_share"`
	DeploymentID    string   `json:"deployment_id"`
	TenantID        string   `json:"tenant_id"`
	LicenseID       string   `json:"license_id"`
}

// HybridRuntime — последнее состояние Relay (не секреты в открытом виде в API).
type HybridRuntime struct {
	LastSyncAt      time.Time `json:"last_sync_at,omitempty"`
	LastLeaseRenew  time.Time `json:"last_lease_renew,omitempty"`
	LeaseStatus     string    `json:"lease_status,omitempty"`
	LeaseMessage    string    `json:"lease_message,omitempty"`
	LastCRLIssuedAt time.Time `json:"last_crl_issued_at,omitempty"`
	LastBundleID    string    `json:"last_bundle_id,omitempty"`
	LastError       string    `json:"last_error,omitempty"`
}

// EgressAuditEntry — журнал исходящих соединений Relay (ADR-0018 §3.3).
type EgressAuditEntry struct {
	ID        string    `json:"id"`
	At        time.Time `json:"at"`
	Kind      string    `json:"kind"`
	Target    string    `json:"target"`
	Level     string    `json:"level"`
	Bytes     int       `json:"bytes"`
	PayloadHash string  `json:"payload_hash"`
}

// DefaultHybridPolicy — connected выключен по умолчанию (air-gap first).
func DefaultHybridPolicy() HybridPolicy {
	return HybridPolicy{
		Enabled:     false,
		HealthLevel: "A",
		TIShare:     false,
	}
}
