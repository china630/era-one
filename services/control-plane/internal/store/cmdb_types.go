package store

import "time"

// Asset — зарегистрированный хост (CMDB current-state, ADR-0011).
type Asset struct {
	NodeID       string    `json:"node_id"`
	TenantID     string    `json:"tenant_id"`
	Hostname     string    `json:"hostname"`
	Platform     string    `json:"platform"`
	AgentID      string    `json:"agent_id"`
	LastSeen     time.Time `json:"last_seen"`
	AgentVersion string    `json:"agent_version"`
	// ITAM fields (optional, backward-compatible)
	FQDN               string    `json:"fqdn,omitempty"`
	OSName             string    `json:"os_name,omitempty"`
	OSVersion          string    `json:"os_version,omitempty"`
	Kernel             string    `json:"kernel,omitempty"`
	CPUModel           string    `json:"cpu_model,omitempty"`
	CPUCores           uint32    `json:"cpu_cores,omitempty"`
	RAMMB              uint64    `json:"ram_mb,omitempty"`
	DiskTotalGB        uint64    `json:"disk_total_gb,omitempty"`
	SerialNumber       string    `json:"serial_number,omitempty"`
	BoardSerial        string    `json:"board_serial,omitempty"`
	Manufacturer       string    `json:"manufacturer,omitempty"`
	Model              string    `json:"model,omitempty"`
	MACAddrs           []string  `json:"mac_addrs,omitempty"`
	IPAddrs            []string  `json:"ip_addrs,omitempty"`
	InventoryUpdatedAt time.Time `json:"inventory_updated_at,omitempty"`
	// Observe / network CMDB (ADR-0020)
	AssetKind string `json:"asset_kind,omitempty"`
	Managed   bool   `json:"managed"`
}

// AssetSoftware — установленное ПО на активе.
type AssetSoftware struct {
	NodeID      string    `json:"node_id"`
	TenantID    string    `json:"tenant_id"`
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Vendor      string    `json:"vendor,omitempty"`
	Source      string    `json:"source,omitempty"`
	InstallDate time.Time `json:"install_date,omitempty"`
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`
}

// Contract — финансовый ITAM: договор с вендором.
type Contract struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Vendor      string    `json:"vendor"`
	Name        string    `json:"name"`
	StartDate   time.Time `json:"start_date"`
	EndDate     time.Time `json:"end_date"`
	CostAnnual  float64   `json:"cost_annual"`
	Currency    string    `json:"currency,omitempty"`
}

// SoftwareLicense — entitlement (лицензии ПО).
type SoftwareLicense struct {
	ID             string `json:"id"`
	TenantID       string `json:"tenant_id"`
	Product        string `json:"product"`
	EntitledSeats  int    `json:"entitled_seats"`
	ContractID     string `json:"contract_id,omitempty"`
}

// ReconcileRow — installed vs entitled.
type ReconcileRow struct {
	Product       string `json:"product"`
	Installed     int    `json:"installed"`
	Entitled      int    `json:"entitled"`
	Delta         int    `json:"delta"` // installed - entitled
	Compliance    string `json:"compliance"` // ok | over | under
}
