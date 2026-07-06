// Package rotation — планировщик ротации секретов PAM (L-03).
package rotation

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"time"

	"era/services/pam/internal/checkout"
	"era/services/pam/internal/vault"
)

// Policy — интервал ротации по target.
type Policy struct {
	Target         string
	Interval       time.Duration
	LastRotated    map[string]time.Time
}

// Scheduler периодически ротирует пароли и инвалидирует checkout.
type Scheduler struct {
	Vault     *vault.Vault
	Checkouts *checkout.Store
	Policy    Policy
	ticker    *time.Ticker
}

func NewScheduler(v *vault.Vault, ch *checkout.Store, interval time.Duration) *Scheduler {
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	return &Scheduler{
		Vault:     v,
		Checkouts: ch,
		Policy:    Policy{Interval: interval, LastRotated: make(map[string]time.Time)},
	}
}

// Run запускает фоновую ротацию до отмены ctx.
func (s *Scheduler) Run(ctx context.Context) {
	s.ticker = time.NewTicker(s.Policy.Interval)
	defer s.ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.ticker.C:
			s.rotateDue()
		}
	}
}

// TickOnce — одна итерация (для тестов).
func (s *Scheduler) TickOnce() int {
	return s.rotateDue()
}

func (s *Scheduler) rotateDue() int {
	if s.Vault == nil || s.Vault.Sealed() {
		return 0
	}
	rotated := 0
	now := time.Now().UTC()
	for _, meta := range s.Vault.ListMeta() {
		last, ok := s.Policy.LastRotated[meta.ID]
		if ok && now.Sub(last) < s.Policy.Interval {
			continue
		}
		pw, err := randomPassword(16)
		if err != nil {
			log.Printf("rotation: random password: %v", err)
			continue
		}
		if err := s.Vault.RotatePassword(meta.ID, pw); err != nil {
			log.Printf("rotation: secret %s: %v", meta.ID, err)
			continue
		}
		s.revokeCheckouts(meta.ID)
		s.Policy.LastRotated[meta.ID] = now
		rotated++
	}
	return rotated
}

func (s *Scheduler) revokeCheckouts(secretID string) {
	if s.Checkouts == nil {
		return
	}
	for _, req := range s.Checkouts.List() {
		if req.SecretID == secretID && (req.Status == checkout.StatusApproved || req.Status == checkout.StatusPending) {
			_ = s.Checkouts.Revoke(req.ID)
		}
	}
}

func randomPassword(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
