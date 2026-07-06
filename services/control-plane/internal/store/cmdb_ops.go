package store

// CmdbOps — CMDB/ITAM операции (ADR-0011).
type CmdbOps interface {
	GetAsset(nodeID string) (*Asset, bool)
	FindAssetByAgentID(agentID string) (*Asset, bool)
	FindAssetBySerial(serial string) (*Asset, bool)
	UpsertAssetFull(a *Asset)
	ReplaceAssetSoftware(nodeID, tenantID string, items []*AssetSoftware)
	ListAssetSoftware(nodeID string) []*AssetSoftware
	ListAllAssetSoftware() []*AssetSoftware
	UpsertContract(c *Contract)
	ListContracts() []*Contract
	UpsertSoftwareLicense(l *SoftwareLicense)
	ListSoftwareLicenses() []*SoftwareLicense
	ReconcileSoftwareLicenses() []ReconcileRow
}
