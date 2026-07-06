package license

import (
	"testing"
	"time"
)

func TestValidatorHappyPath(t *testing.T) {
	pub, priv, _ := GenerateKeypair()
	now := time.Now()
	token, _ := Sign(newClaims(now, 1), priv)

	v := &Validator{Pub: pub, Clock: NewSealedClock([]byte("s"), &memStore{})}
	ev, claims, err := v.Check(token, now.AddDate(0, 1, 0), 10)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Status != StatusValid {
		t.Fatalf("ожидался VALID, получено %s (%s)", ev.Status, ev.Message)
	}
	if claims.Customer != "Test Bank" {
		t.Fatalf("claims не разобраны: %+v", claims)
	}
}

func TestValidatorRevoked(t *testing.T) {
	pub, priv, _ := GenerateKeypair()
	now := time.Now()
	c := newClaims(now, 1)
	c.LicenseID = "lic-revoked"
	token, _ := Sign(c, priv)

	v := &Validator{Pub: pub, CRL: &CRL{Revoked: []string{"lic-revoked"}}}
	ev, _, err := v.Check(token, now, 1)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Status != StatusExpired || !ev.Degraded {
		t.Fatalf("ожидался EXPIRED+degraded для отозванной, получено %s", ev.Status)
	}
}

func TestValidatorDetectsClockRollback(t *testing.T) {
	pub, priv, _ := GenerateKeypair()
	base := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	token, _ := Sign(newClaims(base, 3), priv)

	clock := NewSealedClock([]byte("s"), &memStore{})
	v := &Validator{Pub: pub, Clock: clock}

	// Первая проверка фиксирует high-water на base.
	if _, _, err := v.Check(token, base, 1); err != nil {
		t.Fatal(err)
	}
	// Заказчик отмотал часы назад на год, чтобы «продлить».
	ev, _, err := v.Check(token, base.AddDate(-1, 0, 0), 1)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Status != StatusExpired || !ev.Degraded {
		t.Fatalf("ожидался детект отката (EXPIRED+degraded), получено %s", ev.Status)
	}
}
