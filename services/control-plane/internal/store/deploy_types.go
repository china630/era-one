package store

import "time"

// RolloutStatus — статус доставки пакета на хост.
type RolloutStatus string

const (
	RolloutPending   RolloutStatus = "pending"
	RolloutRunning   RolloutStatus = "running"
	RolloutSucceeded RolloutStatus = "succeeded"
	RolloutFailed    RolloutStatus = "failed"
)

// DeployJob — silent install подписанного пакета (Vision P4).
type DeployJob struct {
	ID          string        `json:"id"`
	TenantID    string        `json:"tenant_id"`
	NodeID      string        `json:"node_id"`
	PackageRef  string        `json:"package_ref"`
	OTAToken    string        `json:"ota_token,omitempty"`
	Reboot      bool          `json:"reboot"`
	Status      RolloutStatus `json:"status"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// PatchJob — patch по CVE-дельте (inventory vs catalog).
type PatchJob struct {
	ID         string        `json:"id"`
	TenantID   string        `json:"tenant_id"`
	NodeID     string        `json:"node_id"`
	CVEID      string        `json:"cve_id"`
	Product    string        `json:"product"`
	PackageRef string        `json:"package_ref"`
	Status     RolloutStatus `json:"status"`
	CreatedAt  time.Time     `json:"created_at"`
}

// PatchPlanRow — строка плана патчей для node.
type PatchPlanRow struct {
	NodeID     string `json:"node_id"`
	Product    string `json:"product"`
	Version    string `json:"version"`
	CVEID      string `json:"cve_id"`
	PackageRef string `json:"package_ref"`
}
