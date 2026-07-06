// Package license — офлайн-лицензирование ERA XDR (ADR-0010).
//
// Модель: вендор подписывает Claims приватным ключом Ed25519; продукт проверяет
// подпись встроенным публичным ключом локально (air-gap, без сети).
//
// Токен: ERA1.<base64url(claims_json)>.<base64url(signature)>
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

// Format — префикс/версия формата токена.
const Format = "ERA1"

// ClaimsVersion — версия структуры Claims.
const ClaimsVersion = 1

// Module — лицензируемый модуль (соответствует изданиям из ADR-0005).
type Module string

const (
	ModuleVuln      Module = "vm"        // ERA Vuln (сканер уязвимостей)
	ModuleControlAI Module = "control-ai" // ERA Control AI (SOC Analyst, hunting)
	ModuleAILegacy    Module = "ai"        // deprecated alias; см. HasModule
	ModuleResponse  Module = "response"  // ERA Response (SOAR)
	ModuleFederated Module = "federated" // ERA Federated (опция)
	ModuleNational  Module = "national"  // ERA National (опция)
)

// KnownModules — все опциональные модули (ERA Core входит в любую лицензию).
var KnownModules = []Module{ModuleVuln, ModuleControlAI, ModuleResponse, ModuleFederated, ModuleNational}

// Claims — подписываемая полезная нагрузка лицензии.
type Claims struct {
	Version    int      `json:"v"`
	LicenseID  string   `json:"lid"`
	Customer   string   `json:"cust"`
	TenantID   string   `json:"tenant"`
	Edition    string   `json:"edition"` // базовое издание, напр. "core"
	Modules    []Module `json:"modules"`
	MaxNodes   int      `json:"max_nodes"`
	Deployment string   `json:"deployment,omitempty"` // привязка к инсталляции
	IssuedAt   int64    `json:"iat"`
	NotBefore  int64    `json:"nbf"`
	ExpiresAt  int64    `json:"exp"`
	GraceDays  int      `json:"grace_days"`
}

// HasModule сообщает, включён ли модуль в лицензии.
func (c *Claims) HasModule(m Module) bool {
	for _, x := range c.Modules {
		if x == m {
			return true
		}
		// Legacy: токены до переименования ERA AI → ERA Control AI (control-ai).
		if m == ModuleControlAI && x == ModuleAILegacy {
			return true
		}
	}
	return false
}

// Sign сериализует Claims и возвращает подписанный токен.
func Sign(c *Claims, priv ed25519.PrivateKey) (string, error) {
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
	sig := ed25519.Sign(priv, payload)
	return strings.Join([]string{
		Format,
		base64.RawURLEncoding.EncodeToString(payload),
		base64.RawURLEncoding.EncodeToString(sig),
	}, "."), nil
}

// Verify проверяет подпись токена публичным ключом и возвращает Claims.
// Проверка ВРЕМЕНИ/лимитов — отдельно через Evaluate (разделение ответственности).
func Verify(token string, pub ed25519.PublicKey) (*Claims, error) {
	if len(pub) != ed25519.PublicKeySize {
		return nil, errors.New("license: некорректный размер публичного ключа")
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] != Format {
		return nil, errors.New("license: неверный формат токена")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("license: decode payload: %w", err)
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("license: decode signature: %w", err)
	}
	if !ed25519.Verify(pub, payload, sig) {
		return nil, errors.New("license: подпись недействительна")
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

// Status — статус лицензии на момент проверки (ADR-0010, §4).
type Status string

const (
	StatusValid             Status = "VALID"
	StatusGrace             Status = "GRACE"
	StatusExpired           Status = "EXPIRED"
	StatusNotYetValid       Status = "NOT_YET_VALID"
	StatusNodeLimitExceeded Status = "NODE_LIMIT_EXCEEDED"
)

// Evaluation — результат оценки лицензии.
type Evaluation struct {
	Status     Status
	Message    string
	DaysLeft   int  // до exp (может быть отрицательным в grace)
	Degraded   bool // продукт должен работать в ограниченном режиме
	AllowNewOnboarding bool
}

// Evaluate оценивает лицензию относительно текущего времени, привязки и числа узлов.
// deployment — отпечаток текущей инсталляции ("" = не проверять привязку).
func (c *Claims) Evaluate(now time.Time, deployment string, activeNodes int) Evaluation {
	// 1) Привязка к развёртыванию.
	if c.Deployment != "" && deployment != "" && c.Deployment != deployment {
		return Evaluation{
			Status:   StatusExpired,
			Message:  "лицензия привязана к другому развёртыванию",
			Degraded: true,
		}
	}

	nbf := time.Unix(c.NotBefore, 0)
	exp := time.Unix(c.ExpiresAt, 0)
	graceEnd := exp.Add(time.Duration(c.GraceDays) * 24 * time.Hour)
	daysLeft := int(exp.Sub(now).Hours() / 24)

	// 2) Время.
	switch {
	case now.Before(nbf):
		return Evaluation{Status: StatusNotYetValid, Message: "лицензия ещё не активна", DaysLeft: daysLeft}
	case now.After(graceEnd):
		return Evaluation{
			Status:   StatusExpired,
			Message:  "срок лицензии истёк (включая grace); режим деградации",
			DaysLeft: daysLeft,
			Degraded: true,
		}
	case now.After(exp):
		// grace-период
		return Evaluation{
			Status:             StatusGrace,
			Message:            fmt.Sprintf("лицензия в grace-периоде, перевыпустите (grace до %s)", graceEnd.Format("2006-01-02")),
			DaysLeft:           daysLeft,
			AllowNewOnboarding: true,
		}
	}

	// 3) Лимит узлов (в пределах срока).
	if c.MaxNodes > 0 && activeNodes > c.MaxNodes {
		return Evaluation{
			Status:   StatusNodeLimitExceeded,
			Message:  fmt.Sprintf("превышен лимит узлов: %d > %d", activeNodes, c.MaxNodes),
			DaysLeft: daysLeft,
			// онбординг новых блокируется, существующие работают
			AllowNewOnboarding: false,
		}
	}

	return Evaluation{
		Status:             StatusValid,
		Message:            "лицензия действительна",
		DaysLeft:           daysLeft,
		AllowNewOnboarding: true,
	}
}
