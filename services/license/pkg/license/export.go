// Package license — публичный API лицензирования ERA (ADR-0010, ADR-0018).
package license

import internal "era/services/license/internal/license"

type (
	LeaseClaims     = internal.LeaseClaims
	LeaseStatus     = internal.LeaseStatus
	LeaseEvaluation = internal.LeaseEvaluation
	CRL             = internal.CRL
	Module          = internal.Module
	Claims          = internal.Claims
	Evaluation      = internal.Evaluation
	Status          = internal.Status
	Validator       = internal.Validator
	SealedClock     = internal.SealedClock
	FileClockStore  = internal.FileClockStore
)

const (
	LeaseFormat         = internal.LeaseFormat
	LeaseClaimsVersion  = internal.LeaseClaimsVersion
	LeaseStatusValid    = internal.LeaseStatusValid
	LeaseStatusGrace    = internal.LeaseStatusGrace
	LeaseStatusExpired  = internal.LeaseStatusExpired
	LeaseStatusMismatch = internal.LeaseStatusMismatch
	StatusValid         = internal.StatusValid
	StatusGrace         = internal.StatusGrace
	StatusExpired       = internal.StatusExpired
)

var (
	GenerateKeypair    = internal.GenerateKeypair
	EncodeKey          = internal.EncodeKey
	DecodePublicKey    = internal.DecodePublicKey
	DecodePrivateKey   = internal.DecodePrivateKey
	SignLease          = internal.SignLease
	VerifyLease        = internal.VerifyLease
	SignCRL            = internal.SignCRL
	VerifyCRL          = internal.VerifyCRL
	DefaultLeasePolicy = internal.DefaultLeasePolicy
	Sign               = internal.Sign
	Verify             = internal.Verify
	NewSealedClock     = internal.NewSealedClock
)
