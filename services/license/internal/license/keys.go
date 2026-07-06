package license

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
)

// GenerateKeypair создаёт новую пару ключей вендора Ed25519.
func GenerateKeypair() (pub ed25519.PublicKey, priv ed25519.PrivateKey, err error) {
	return ed25519.GenerateKey(rand.Reader)
}

// EncodeKey кодирует ключ (приватный или публичный) в base64 (std).
func EncodeKey(key []byte) string {
	return base64.StdEncoding.EncodeToString(key)
}

// DecodePublicKey разбирает публичный ключ из base64.
func DecodePublicKey(s string) (ed25519.PublicKey, error) {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("license: decode public key: %w", err)
	}
	if len(b) != ed25519.PublicKeySize {
		return nil, errors.New("license: некорректный размер публичного ключа")
	}
	return ed25519.PublicKey(b), nil
}

// DecodePrivateKey разбирает приватный ключ из base64.
func DecodePrivateKey(s string) (ed25519.PrivateKey, error) {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("license: decode private key: %w", err)
	}
	if len(b) != ed25519.PrivateKeySize {
		return nil, errors.New("license: некорректный размер приватного ключа")
	}
	return ed25519.PrivateKey(b), nil
}
