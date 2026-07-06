// Package bundle — подписанные контент-бандлы ERA Update Service (ADR-0018 §1.1.1).
package bundle

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	lic "era/services/license/pkg/license"
)

// Format — префикс wire-токена бандла.
const Format = "ERABNDL1"

// Поддерживаемые kind бандлов (ADR-0018 §1.1.1).
const (
	KindSigmaCorpus = "sigma-corpus"
	KindCVEFeed     = "cve-feed"
	KindConnector   = "connector"
	KindAIPack      = "ai-pack"
)

// ValidKind проверяет известный kind.
func ValidKind(kind string) bool {
	switch kind {
	case KindSigmaCorpus, KindCVEFeed, KindConnector, KindAIPack:
		return true
	default:
		return false
	}
}

// ClaimsVersion — версия структуры Claims.
const ClaimsVersion = 1

// Claims — подписываемая метаинформация бандла (без сырого контента в токене).
type Claims struct {
	Version      int    `json:"v"`
	BundleID     string `json:"id"`
	Kind         string `json:"kind"`
	IssuedAt     int64  `json:"iat"`
	FileCount    int    `json:"file_count"`
	ManifestHash string `json:"manifest_hash"`
}

// FileEntry — элемент манифеста.
type FileEntry struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

// Manifest — список файлов в бандле.
type Manifest struct {
	Files []FileEntry `json:"files"`
}

// SignClaims подписывает claims и возвращает токен ERABNDL1.
func SignClaims(c *Claims, priv ed25519.PrivateKey) (string, error) {
	if c.Version == 0 {
		c.Version = ClaimsVersion
	}
	payload, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	sig := ed25519.Sign(priv, payload)
	return strings.Join([]string{
		Format,
		base64.RawURLEncoding.EncodeToString(payload),
		base64.RawURLEncoding.EncodeToString(sig),
	}, "."), nil
}

// Verify проверяет токен и возвращает claims.
func Verify(token string, pub ed25519.PublicKey) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] != Format {
		return nil, errors.New("bundle: неверный формат")
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
		return nil, errors.New("bundle: подпись недействительна")
	}
	var c Claims
	if err := json.Unmarshal(payload, &c); err != nil {
		return nil, err
	}
	if c.Version != ClaimsVersion {
		return nil, fmt.Errorf("bundle: unsupported version %d", c.Version)
	}
	return &c, nil
}

// HashManifest вычисляет стабильный хэш манифеста.
func HashManifest(m *Manifest) string {
	b, _ := json.Marshal(m)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:8])
}

// LoadSigningKey из env ERA_UPDATE_SIGN_KEY (base64) или файла ERA_UPDATE_SIGN_KEY_FILE.
func LoadSigningKey() (ed25519.PrivateKey, ed25519.PublicKey, error) {
	raw := os.Getenv("ERA_UPDATE_SIGN_KEY")
	if raw == "" {
		if p := os.Getenv("ERA_UPDATE_SIGN_KEY_FILE"); p != "" {
			b, err := os.ReadFile(p)
			if err != nil {
				return nil, nil, err
			}
			raw = strings.TrimSpace(string(b))
		}
	}
	if raw == "" {
		pub, priv, err := ed25519.GenerateKey(nil)
		return priv, pub, err
	}
	priv, err := lic.DecodePrivateKey(raw)
	if err != nil {
		return nil, nil, err
	}
	return priv, priv.Public().(ed25519.PublicKey), nil
}

// NewClaims создаёт claims для sigma-corpus бандла.
func NewClaims(kind string, m *Manifest) *Claims {
	return &Claims{
		BundleID:     fmt.Sprintf("bnd-%d", time.Now().UTC().Unix()),
		Kind:         kind,
		IssuedAt:     time.Now().UTC().Unix(),
		FileCount:    len(m.Files),
		ManifestHash: HashManifest(m),
	}
}
