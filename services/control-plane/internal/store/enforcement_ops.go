package store

// EnforcementOps — policy bundle + BitLocker escrow (Stage 6).
type EnforcementOps interface {
	GetEnforcementPolicy() EnforcementPolicy
	SetEnforcementPolicy(p EnforcementPolicy, actor, detail string) error
	RollbackEnforcementPolicy(actor string) (EnforcementPolicy, bool)
	ListEnforcementHistory(limit int) []EnforcementPolicyRevision
	UpsertBitlockerEscrow(e *BitlockerEscrow)
	GetBitlockerEscrow(nodeID, volumeID string) (*BitlockerEscrow, bool)
	ListBitlockerEscrows(nodeID string) []BitlockerEscrowPublic
}
