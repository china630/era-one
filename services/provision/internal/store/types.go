package store

import "time"

// OSImage — образ в локальном репозитории (MinIO bucket ref).
type OSImage struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Platform    string    `json:"platform"`
	Version     string    `json:"version"`
	MinIORef    string    `json:"minio_ref"`
	Unattended  string    `json:"unattended_kind"` // kickstart | preseed | autounattend
	CreatedAt   time.Time `json:"created_at"`
}

// PXEConfig — boot-конфигурация для стенда (TFTP/PXE — simulated в MVP).
type PXEConfig struct {
	TFTPRoot   string            `json:"tftp_root"`
	DefaultImage string          `json:"default_image"`
	BootMenu   []PXEBootEntry    `json:"boot_menu"`
}

type PXEBootEntry struct {
	Label   string `json:"label"`
	ImageID string `json:"image_id"`
	Kernel  string `json:"kernel"`
	Initrd  string `json:"initrd,omitempty"`
}

// EnrollRequest — post-install регистрация хоста в CMDB.
type EnrollRequest struct {
	AgentID      string `json:"agent_id"`
	TenantID     string `json:"tenant_id"`
	NodeID       string `json:"node_id"`
	Hostname     string `json:"hostname"`
	Platform     string `json:"platform"`
	AgentVersion string `json:"agent_version"`
	ImageID      string `json:"image_id,omitempty"`
}
