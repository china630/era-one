package hybrid

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// bundleClaims — минимальный разбор для Relay (полный формат в update-service).
type bundleClaims struct {
	BundleID string `json:"id"`
	Kind     string `json:"kind"`
}

const bundleFormat = "ERABNDL1"

// minimalBundleVerify — проверка ERABNDL1 (wire-формат update-service).
func minimalBundleVerify(token string, pub ed25519.PublicKey) (*bundleClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] != bundleFormat {
		return nil, fmt.Errorf("bundle: неверный формат")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("bundle: decode payload: %w", err)
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("bundle: decode sig: %w", err)
	}
	if !ed25519.Verify(pub, payload, sig) {
		return nil, fmt.Errorf("bundle: подпись недействительна")
	}
	var c bundleClaims
	if err := json.Unmarshal(payload, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func verifyBundleToken(token string, pub ed25519.PublicKey) (*bundleClaims, error) {
	return minimalBundleVerify(token, pub)
}
