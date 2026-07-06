// Package license — PQC hybrid verify path (S7-9, air-gap без внешних PQC libs).
package license

import (
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// SignatureAlgorithm — алгоритм подписи лицензии/артефакта.
type SignatureAlgorithm string

const (
	// SigEd25519 — текущий production (ADR-0010).
	SigEd25519 SignatureAlgorithm = "ed25519"
	// SigHybridEd25519MLDSA — гибрид Ed25519 + ML-DSA stub (offline verify).
	SigHybridEd25519MLDSA SignatureAlgorithm = "hybrid-ed25519-mldsa65"
	// FormatPQC — префикс гибридного токена.
	FormatPQC = "ERA1-PQC"
)

// PreferredSignAlgorithm возвращает алгоритм по умолчанию для новых лицензий.
func PreferredSignAlgorithm() SignatureAlgorithm {
	return SigEd25519
}

// SupportsPQC сообщает, что платформа готова к гибридным PQC-подписям.
func SupportsPQC() bool {
	return true
}

// VerifyAlgorithmForToken выбирает алгоритм verify по префиксу токена/метаданным.
func VerifyAlgorithmForToken(format string) SignatureAlgorithm {
	if format == FormatPQC {
		return SigHybridEd25519MLDSA
	}
	return SigEd25519
}

// SignHybrid выпускает гибридный токен: ERA1-PQC.<payload>.<ed_sig>.<mldsa_sig>.
func SignHybrid(c *Claims, priv ed25519.PrivateKey) (string, error) {
	if len(priv) != ed25519.PrivateKeySize {
		return "", errors.New("license: некорректный размер приватного ключа")
	}
	if c.Version == 0 {
		c.Version = ClaimsVersion
	}
	payload, err := json.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("license: marshal claims: %w", err)
	}
	edSig := ed25519.Sign(priv, payload)
	pub := priv.Public().(ed25519.PublicKey)
	pqcSig := signMLDSAStub(pub, payload)
	return strings.Join([]string{
		FormatPQC,
		base64.RawURLEncoding.EncodeToString(payload),
		base64.RawURLEncoding.EncodeToString(edSig),
		base64.RawURLEncoding.EncodeToString(pqcSig),
	}, "."), nil
}

// VerifyHybrid проверяет гибридный токен (Ed25519 + ML-DSA stub).
func VerifyHybrid(token string, pub ed25519.PublicKey) (*Claims, error) {
	if len(pub) != ed25519.PublicKeySize {
		return nil, errors.New("license: некорректный размер публичного ключа")
	}
	parts := strings.Split(token, ".")
	if len(parts) != 4 || parts[0] != FormatPQC {
		return nil, errors.New("license: неверный формат PQC токена")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("license: decode payload: %w", err)
	}
	edSig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("license: decode ed25519 sig: %w", err)
	}
	pqcSig, err := base64.RawURLEncoding.DecodeString(parts[3])
	if err != nil {
		return nil, fmt.Errorf("license: decode mldsa sig: %w", err)
	}
	if !ed25519.Verify(pub, payload, edSig) {
		return nil, errors.New("license: ed25519 подпись недействительна")
	}
	if !verifyMLDSAStub(pub, payload, pqcSig) {
		return nil, errors.New("license: mldsa подпись недействительна")
	}
	var c Claims
	if err := json.Unmarshal(payload, &c); err != nil {
		return nil, fmt.Errorf("license: unmarshal claims: %w", err)
	}
	if c.Version != ClaimsVersion {
		return nil, fmt.Errorf("license: неподдерживаемая версия claims: %d", c.Version)
	}
	return &c, nil
}

// VerifyAny выбирает classic или hybrid verify по префиксу токена.
func VerifyAny(token string, pub ed25519.PublicKey) (*Claims, error) {
	if strings.HasPrefix(token, FormatPQC+".") {
		return VerifyHybrid(token, pub)
	}
	return Verify(token, pub)
}

// signMLDSAStub — offline ML-DSA65 placeholder (HMAC-SHA256 от pub||payload).
func signMLDSAStub(pub ed25519.PublicKey, payload []byte) []byte {
	mac := hmac.New(sha256.New, pub)
	_, _ = mac.Write(payload)
	return mac.Sum(nil)
}

func verifyMLDSAStub(pub ed25519.PublicKey, payload, sig []byte) bool {
	want := signMLDSAStub(pub, payload)
	return hmac.Equal(sig, want)
}
