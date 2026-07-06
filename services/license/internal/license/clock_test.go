package license

import (
	"testing"
	"time"
)

type memStore struct{ data []byte }

func (m *memStore) Load() ([]byte, error) { return m.data, nil }
func (m *memStore) Save(d []byte) error   { m.data = append([]byte{}, d...); return nil }

func TestSealedClockAdvances(t *testing.T) {
	c := NewSealedClock([]byte("install-secret"), &memStore{})
	base := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	eff, rollback, err := c.EffectiveNow(base)
	if err != nil || rollback {
		t.Fatalf("неожиданный откат/ошибка: %v rollback=%v", err, rollback)
	}
	if !eff.Equal(base) {
		t.Fatalf("ожидалось %v, получено %v", base, eff)
	}
}

func TestSealedClockDetectsRollback(t *testing.T) {
	st := &memStore{}
	c := NewSealedClock([]byte("install-secret"), st)
	base := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	// Зафиксировали high-water на base.
	if _, _, err := c.EffectiveNow(base); err != nil {
		t.Fatal(err)
	}
	// Заказчик отмотал часы на месяц назад.
	past := base.AddDate(0, -1, 0)
	eff, rollback, err := c.EffectiveNow(past)
	if err != nil {
		t.Fatal(err)
	}
	if !rollback {
		t.Fatal("ожидался детект отката часов")
	}
	// Эффективное время не должно уходить назад (монотонность).
	if eff.Before(base) {
		t.Fatalf("эффективное время ушло назад: %v < %v", eff, base)
	}
}

func TestSealedClockDetectsTamper(t *testing.T) {
	st := &memStore{data: []byte(`{"hw":9999999999,"mac":"deadbeef"}`)}
	c := NewSealedClock([]byte("install-secret"), st)
	_, rollback, err := c.EffectiveNow(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if !rollback {
		t.Fatal("ожидался детект подделки запечатанного состояния")
	}
}
