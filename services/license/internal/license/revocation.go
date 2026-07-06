package license

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// CRLFormat — префикс/версия формата списка отзыва.
const CRLFormat = "ERACRL1"

// CRL — подписанный вендором список отозванных лицензий (ADR-0010 §6).
type CRL struct {
	Version  int      `json:"v"`
	IssuedAt int64    `json:"iat"`
	Revoked  []string `json:"revoked"` // список license_id (lid)
}

// SignCRL сериализует и подписывает список отзыва.
func SignCRL(crl *CRL, priv ed25519.PrivateKey) (string, error) {
	if len(priv) != ed25519.PrivateKeySize {
		return "", errors.New("crl: некорректный размер приватного ключа")
	}
	if crl.Version == 0 {
		crl.Version = 1
	}
	payload, err := json.Marshal(crl)
	if err != nil {
		return "", fmt.Errorf("crl: marshal: %w", err)
	}
	sig := ed25519.Sign(priv, payload)
	return strings.Join([]string{
		CRLFormat,
		base64.RawURLEncoding.EncodeToString(payload),
		base64.RawURLEncoding.EncodeToString(sig),
	}, "."), nil
}

// VerifyCRL проверяет подпись списка отзыва публичным ключом.
func VerifyCRL(token string, pub ed25519.PublicKey) (*CRL, error) {
	if len(pub) != ed25519.PublicKeySize {
		return nil, errors.New("crl: некорректный размер публичного ключа")
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] != CRLFormat {
		return nil, errors.New("crl: неверный формат токена")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("crl: decode payload: %w", err)
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("crl: decode signature: %w", err)
	}
	if !ed25519.Verify(pub, payload, sig) {
		return nil, errors.New("crl: подпись недействительна")
	}
	var c CRL
	if err := json.Unmarshal(payload, &c); err != nil {
		return nil, fmt.Errorf("crl: unmarshal: %w", err)
	}
	return &c, nil
}

// IsRevoked сообщает, отозвана ли лицензия с данным lid.
func (c *CRL) IsRevoked(lid string) bool {
	for _, r := range c.Revoked {
		if r == lid {
			return true
		}
	}
	return false
}
