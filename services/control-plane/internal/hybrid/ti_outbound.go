package hybrid

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// PseudonymizeTIOutbound — L-02: псевдонимизация исходящего TI payload (без PII).
func PseudonymizeTIOutbound(raw string, salt string) string {
	if salt == "" {
		salt = "era-ti-salt"
	}
	h := sha256.Sum256([]byte(salt + "|" + strings.TrimSpace(raw)))
	return "ti-" + hex.EncodeToString(h[:8])
}

// TIOutboundAuditEntry — метаданные egress без сырого IOC.
type TIOutboundAuditEntry struct {
	Kind        string `json:"kind"`
	PseudonymID string `json:"pseudonym_id"`
	Bytes       int    `json:"bytes"`
}

func BuildTIOutboundAudit(kind, raw string, salt string) TIOutboundAuditEntry {
	return TIOutboundAuditEntry{
		Kind:        kind,
		PseudonymID: PseudonymizeTIOutbound(raw, salt),
		Bytes:       len(raw),
	}
}
