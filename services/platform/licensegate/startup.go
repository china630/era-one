// Startup license validation (ADR-0010, P0-1).
package licensegate

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	lic "era/services/license/pkg/license"
)

// StrictMode — fail-closed: ERA_LICENSE_STRICT=1|true или ERA_PRODUCTION=1|true.
func StrictMode() bool {
	return envTruthy("ERA_LICENSE_STRICT") || envTruthy("ERA_PRODUCTION")
}

// GateFromEnv строит Gate из проверенного лицензионного токена или DevDefault в dev.
func GateFromEnv(activeNodes int) (*Gate, error) {
	token, err := loadLicenseToken()
	if err != nil {
		if StrictMode() {
			return nil, err
		}
		return DevDefault(), nil
	}
	if token == "" {
		if StrictMode() {
			return nil, errors.New("licensegate: ERA_LICENSE_TOKEN/ERA_LICENSE_PATH обязателен в strict mode")
		}
		return DevDefault(), nil
	}
	ev, claims, err := validateToken(token, activeNodes)
	if err != nil {
		if StrictMode() {
			return nil, fmt.Errorf("licensegate: %w", err)
		}
		return DevDefault(), nil
	}
	if ev.Status != lic.StatusValid && ev.Status != lic.StatusGrace {
		if StrictMode() {
			return nil, fmt.Errorf("licensegate: лицензия %s: %s", ev.Status, ev.Message)
		}
		return DevDefault(), nil
	}
	return gateFromClaims(claims), nil
}

// ValidateStartup проверяет лицензию при старте сервиса (ingest-gateway и др.).
func ValidateStartup(activeNodes int) error {
	token, err := loadLicenseToken()
	if err != nil {
		if StrictMode() {
			return err
		}
		return nil
	}
	if token == "" {
		if StrictMode() {
			return errors.New("license: ERA_LICENSE_TOKEN/ERA_LICENSE_PATH обязателен в strict mode")
		}
		return nil
	}
	ev, _, err := validateToken(token, activeNodes)
	if err != nil {
		return err
	}
	switch ev.Status {
	case lic.StatusValid, lic.StatusGrace:
		return nil
	default:
		return fmt.Errorf("license: %s — %s", ev.Status, ev.Message)
	}
}

func validateToken(token string, activeNodes int) (lic.Evaluation, *lic.Claims, error) {
	pub, err := loadVendorPub()
	if err != nil {
		return lic.Evaluation{}, nil, err
	}
	v := &lic.Validator{
		Pub:        pub,
		Deployment: os.Getenv("ERA_DEPLOYMENT_FINGERPRINT"),
	}
	if path := os.Getenv("ERA_SEALED_CLOCK_PATH"); path != "" {
		secret := []byte(os.Getenv("ERA_SEALED_CLOCK_SECRET"))
		if len(secret) == 0 {
			secret = []byte("era-sealed-clock-dev")
		}
		v.Clock = lic.NewSealedClock(secret, lic.FileClockStore{Path: path})
	}
	return v.Check(token, time.Now().UTC(), activeNodes)
}

func loadLicenseToken() (string, error) {
	if t := strings.TrimSpace(os.Getenv("ERA_LICENSE_TOKEN")); t != "" {
		return t, nil
	}
	path := strings.TrimSpace(os.Getenv("ERA_LICENSE_PATH"))
	if path == "" {
		return "", nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("licensegate: read ERA_LICENSE_PATH: %w", err)
	}
	return strings.TrimSpace(string(b)), nil
}

func loadVendorPub() (ed25519.PublicKey, error) {
	raw := strings.TrimSpace(os.Getenv("ERA_VENDOR_PUB"))
	if raw == "" {
		if path := strings.TrimSpace(os.Getenv("ERA_VENDOR_PUB_FILE")); path != "" {
			b, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("licensegate: read ERA_VENDOR_PUB_FILE: %w", err)
			}
			raw = strings.TrimSpace(string(b))
		}
	}
	if raw == "" {
		return nil, errors.New("licensegate: ERA_VENDOR_PUB или ERA_VENDOR_PUB_FILE обязателен при наличии токена")
	}
	return lic.DecodePublicKey(raw)
}

func gateFromClaims(c *lic.Claims) *Gate {
	var mods []Module
	for _, m := range c.Modules {
		mods = append(mods, Module(m))
	}
	return FromModules(mods)
}

func envTruthy(k string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(k)))
	return v == "1" || v == "true" || v == "yes"
}
