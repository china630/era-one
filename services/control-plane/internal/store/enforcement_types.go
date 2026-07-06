package store

import "time"

// EnforcementPolicy — bundle для агентов (ADR-0012), JSON совместим с era-agent-core.
type EnforcementPolicy struct {
	Version        string              `json:"version"`
	Mode           string              `json:"mode"`
	FailMode       string              `json:"fail_mode"`
	AppRules       []EnforcementAppRule  `json:"app_rules"`
	DeviceRules    []EnforcementDeviceRule `json:"device_rules"`
	VirtualPatches []VirtualPatchRule  `json:"virtual_patches"`
}

type EnforcementAppRule struct {
	ID         string `json:"id"`
	Action     string `json:"action"`
	Path       string `json:"path,omitempty"`
	HashSHA256 string `json:"hash_sha256,omitempty"`
	Signer     string `json:"signer,omitempty"`
	ParentPath string `json:"parent_path,omitempty"`
}

type EnforcementDeviceRule struct {
	ID          string `json:"id"`
	Action      string `json:"action"`
	DeviceClass string `json:"device_class"`
}

type VirtualPatchRule struct {
	ID         string `json:"id"`
	CVEID      string `json:"cve_id"`
	Action     string `json:"action"`
	Path       string `json:"path,omitempty"`
	Vector     string `json:"vector,omitempty"`
	HashSHA256 string `json:"hash_sha256,omitempty"`
}

// EnforcementPolicyRevision — снимок для rollback/audit.
type EnforcementPolicyRevision struct {
	Version   string    `json:"version"`
	Mode      string    `json:"mode"`
	Actor     string    `json:"actor,omitempty"`
	Detail    string    `json:"detail,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// BitlockerEscrow — recovery key escrow (ADR-0009: не логировать ключи).
type BitlockerEscrow struct {
	NodeID    string    `json:"node_id"`
	TenantID  string    `json:"tenant_id"`
	VolumeID  string    `json:"volume_id"`
	KeyBlob   string    `json:"key_blob,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	Actor     string    `json:"actor,omitempty"`
}

// BitlockerEscrowPublic — ответ API без открытого ключа в списках.
type BitlockerEscrowPublic struct {
	NodeID    string    `json:"node_id"`
	TenantID  string    `json:"tenant_id"`
	VolumeID  string    `json:"volume_id"`
	HasKey    bool      `json:"has_key"`
	CreatedAt time.Time `json:"created_at"`
	Actor     string    `json:"actor,omitempty"`
}

func (e *BitlockerEscrow) Public() BitlockerEscrowPublic {
	return BitlockerEscrowPublic{
		NodeID:    e.NodeID,
		TenantID:  e.TenantID,
		VolumeID:  e.VolumeID,
		HasKey:    e.KeyBlob != "",
		CreatedAt: e.CreatedAt,
		Actor:     e.Actor,
	}
}
