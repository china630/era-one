package license

import (
	"crypto/ed25519"
	"time"
)

// Validator — высокоуровневая проверка лицензии на стороне продукта (control-plane).
// Связывает: подпись (Verify) + отзыв (CRL) + anti-rollback (SealedClock) +
// привязку к развёртыванию + лимит узлов.
type Validator struct {
	Pub        ed25519.PublicKey
	Clock      *SealedClock // nil => локальное время (ТОЛЬКО dev, без anti-rollback)
	CRL        *CRL         // nil => без проверки отзыва
	Deployment string       // отпечаток текущей инсталляции ("" => без привязки)
}

// Check выполняет полную проверку и возвращает оценку + разобранные claims.
func (v *Validator) Check(token string, localNow time.Time, activeNodes int) (Evaluation, *Claims, error) {
	claims, err := Verify(token, v.Pub)
	if err != nil {
		return Evaluation{}, nil, err
	}

	// 1) Отзыв (CRL).
	if v.CRL != nil && v.CRL.IsRevoked(claims.LicenseID) {
		return Evaluation{
			Status:   StatusExpired,
			Message:  "лицензия отозвана (CRL)",
			Degraded: true,
		}, claims, nil
	}

	// 2) Anti-rollback: эффективное время вместо локальных часов.
	now := localNow
	if v.Clock != nil {
		// Дата выпуска лицензии — доверенный источник для high-water.
		_ = v.Clock.Observe(time.Unix(claims.IssuedAt, 0))
		eff, rollback, cerr := v.Clock.EffectiveNow(localNow)
		if cerr == nil {
			now = eff
		}
		if rollback {
			return Evaluation{
				Status:   StatusExpired,
				Message:  "обнаружен откат системных часов (anti-rollback); режим деградации",
				Degraded: true,
			}, claims, nil
		}
	}

	// 3) Срок, привязка, лимит узлов.
	return claims.Evaluate(now, v.Deployment, activeNodes), claims, nil
}

// CheckWithLease выполняет полную проверку лицензии + lease (connected-режим, ADR-0018 §6).
// leaseToken пустой => только офлайн-лицензия (air-gap).
func (v *Validator) CheckWithLease(
	token, leaseToken string,
	localNow, lastLeaseRenew time.Time,
	activeNodes int,
) (Evaluation, *Claims, LeaseEvaluation, *LeaseClaims, error) {
	ev, claims, err := v.Check(token, localNow, activeNodes)
	if err != nil {
		return Evaluation{}, nil, LeaseEvaluation{}, nil, err
	}
	if leaseToken == "" {
		return ev, claims, LeaseEvaluation{Status: LeaseStatusValid, Message: "air-gap: lease не требуется"}, nil, nil
	}
	lev, lclaims, lerr := CheckLease(leaseToken, v.Pub, claims.LicenseID, v.Deployment, localNow, lastLeaseRenew)
	if lerr != nil {
		return ev, claims, LeaseEvaluation{}, nil, lerr
	}
	return ev, claims, lev, lclaims, nil
}
