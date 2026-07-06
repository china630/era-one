// Package license — lease поверх офлайн-лицензии (ADR-0018 §6, ADR-0010).
//
// Lease — короткоживущий подписанный entitlement для connected-режима.
// Базовая лицензия (ERA1) остаётся источником истины; lease обновляется через Relay.
package license

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// LeaseFormat — префикс токена lease (отдельно от ERA1 лицензии).
const LeaseFormat = "ERAL1"

// LeaseClaimsVersion — версия структуры LeaseClaims.
const LeaseClaimsVersion = 1

// Degradation flags (не хардкод в логике — задаются в lease/policy, ADR-0018 §6.2).
const (
	DegradeNoNewNodes = "no_new_nodes"
	DegradeNoUpdates  = "no_updates"
)

// LeaseClaims — подписываемая полезная нагрузка lease.
type LeaseClaims struct {
	Version              int      `json:"v"`
	LicenseID            string   `json:"lid"`
	DeploymentID         string   `json:"deployment"`
	TenantID             string   `json:"tenant"`
	Modules              []Module `json:"modules,omitempty"`
	IssuedAt             int64    `json:"iat"`
	ExpiresAt            int64    `json:"exp"`
	GraceDays            int      `json:"grace_days"`
	OfflineMaxDays       int      `json:"offline_max_days"`
	RenewalIntervalHours int      `json:"renewal_interval_h"`
	DegradationMode      []string `json:"degradation_mode,omitempty"`
}

// DefaultLeasePolicy — стартовые значения из editions-control.yaml (не константы рантайма).
func DefaultLeasePolicy() LeaseClaims {
	return LeaseClaims{
		GraceDays:            30,
		OfflineMaxDays:       90,
		RenewalIntervalHours: 24,
		DegradationMode:      []string{DegradeNoNewNodes, DegradeNoUpdates},
	}
}

// LeaseStatus — статус lease на момент проверки.
type LeaseStatus string

const (
	LeaseStatusValid   LeaseStatus = "VALID"
	LeaseStatusGrace   LeaseStatus = "GRACE"
	LeaseStatusExpired LeaseStatus = "EXPIRED"
	LeaseStatusMismatch LeaseStatus = "MISMATCH"
)

// LeaseEvaluation — результат оценки lease (без kill-switch: детекция продолжает работать).
type LeaseEvaluation struct {
	Status             LeaseStatus
	Message            string
	DaysLeft           int
	Degraded           bool
	AllowNewOnboarding bool
	AllowUpdates       bool
}

// SignLease сериализует LeaseClaims и возвращает подписанный токен.
func SignLease(c *LeaseClaims, priv ed25519.PrivateKey) (string, error) {
	if len(priv) != ed25519.PrivateKeySize {
		return "", errors.New("lease: некорректный размер приватного ключа")
	}
	if c.Version == 0 {
		c.Version = LeaseClaimsVersion
	}
	payload, err := json.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("lease: marshal claims: %w", err)
	}
	sig := ed25519.Sign(priv, payload)
	return strings.Join([]string{
		LeaseFormat,
		base64.RawURLEncoding.EncodeToString(payload),
		base64.RawURLEncoding.EncodeToString(sig),
	}, "."), nil
}

// VerifyLease проверяет подпись lease-токена.
func VerifyLease(token string, pub ed25519.PublicKey) (*LeaseClaims, error) {
	if len(pub) != ed25519.PublicKeySize {
		return nil, errors.New("lease: некорректный размер публичного ключа")
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] != LeaseFormat {
		return nil, errors.New("lease: неверный формат токена")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("lease: decode payload: %w", err)
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("lease: decode signature: %w", err)
	}
	if !ed25519.Verify(pub, payload, sig) {
		return nil, errors.New("lease: подпись недействительна")
	}
	var c LeaseClaims
	if err := json.Unmarshal(payload, &c); err != nil {
		return nil, fmt.Errorf("lease: unmarshal claims: %w", err)
	}
	if c.Version != LeaseClaimsVersion {
		return nil, fmt.Errorf("lease: неподдерживаемая версия claims: %d", c.Version)
	}
	return &c, nil
}

// EvaluateLease оценивает lease относительно времени и последнего успешного renew.
// lastRenew — когда Relay/CP последний раз получил валидный lease (zero => только exp/grace).
func (c *LeaseClaims) EvaluateLease(now time.Time, baseLicenseID, deploymentID string, lastRenew time.Time) LeaseEvaluation {
	if baseLicenseID != "" && c.LicenseID != baseLicenseID {
		return LeaseEvaluation{
			Status:             LeaseStatusMismatch,
			Message:            "lease не соответствует license_id",
			Degraded:           true,
			AllowNewOnboarding: false,
			AllowUpdates:       false,
		}
	}
	if deploymentID != "" && c.DeploymentID != "" && c.DeploymentID != deploymentID {
		return LeaseEvaluation{
			Status:             LeaseStatusMismatch,
			Message:            "lease привязан к другому развёртыванию",
			Degraded:           true,
			AllowNewOnboarding: false,
			AllowUpdates:       false,
		}
	}

	exp := time.Unix(c.ExpiresAt, 0)
	graceEnd := exp.Add(time.Duration(c.GraceDays) * 24 * time.Hour)
	daysLeft := int(exp.Sub(now).Hours() / 24)

	allowNodes := true
	allowUpdates := true
	for _, d := range c.DegradationMode {
		switch d {
		case DegradeNoNewNodes:
			// применяется при деградации
		case DegradeNoUpdates:
		}
		_ = d
	}

	// Offline max: слишком долго без успешного renew → деградация (детекция работает).
	if c.OfflineMaxDays > 0 && !lastRenew.IsZero() {
		offlineDeadline := lastRenew.Add(time.Duration(c.OfflineMaxDays) * 24 * time.Hour)
		if now.After(offlineDeadline) {
			allowNodes, allowUpdates = applyDegradation(c.DegradationMode, false, false)
			return LeaseEvaluation{
				Status:             LeaseStatusGrace,
				Message:            fmt.Sprintf("offline max %d дней без renew; режим деградации", c.OfflineMaxDays),
				DaysLeft:           daysLeft,
				Degraded:           true,
				AllowNewOnboarding: allowNodes,
				AllowUpdates:       allowUpdates,
			}
		}
	}

	switch {
	case now.After(graceEnd):
		allowNodes, allowUpdates = applyDegradation(c.DegradationMode, false, false)
		return LeaseEvaluation{
			Status:             LeaseStatusExpired,
			Message:            "lease истёк (включая grace); режим деградации",
			DaysLeft:           daysLeft,
			Degraded:           true,
			AllowNewOnboarding: allowNodes,
			AllowUpdates:       allowUpdates,
		}
	case now.After(exp):
		return LeaseEvaluation{
			Status:             LeaseStatusGrace,
			Message:            fmt.Sprintf("lease в grace до %s", graceEnd.Format("2006-01-02")),
			DaysLeft:           daysLeft,
			Degraded:           false,
			AllowNewOnboarding: true,
			AllowUpdates:       true,
		}
	default:
		return LeaseEvaluation{
			Status:             LeaseStatusValid,
			Message:            "lease действителен",
			DaysLeft:           daysLeft,
			Degraded:           false,
			AllowNewOnboarding: true,
			AllowUpdates:       true,
		}
	}
}

func applyDegradation(modes []string, allowNodes, allowUpdates bool) (bool, bool) {
	for _, m := range modes {
		switch m {
		case DegradeNoNewNodes:
			allowNodes = false
		case DegradeNoUpdates:
			allowUpdates = false
		}
	}
	return allowNodes, allowUpdates
}

// CheckLease — полная проверка lease (подпись + привязка + срок).
func CheckLease(token string, pub ed25519.PublicKey, baseLicenseID, deploymentID string, localNow, lastRenew time.Time) (LeaseEvaluation, *LeaseClaims, error) {
	claims, err := VerifyLease(token, pub)
	if err != nil {
		return LeaseEvaluation{}, nil, err
	}
	return claims.EvaluateLease(localNow, baseLicenseID, deploymentID, lastRenew), claims, nil
}
