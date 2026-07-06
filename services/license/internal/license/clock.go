package license

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

// SealedClock — запечатанные монотонные часы (anti-rollback, ADR-0010 §8).
//
// Хранит «high-water mark» — максимальное достоверное время, скреплённое HMAC.
// Эффективное время = max(local, high_water); откат локальных часов назад
// детектируется. В проде секрет/хранилище должны жить в TPM/OS keystore.
type SealedClock struct {
	secret []byte
	store  ClockStore
	skew   time.Duration
}

// ClockStore — абстракция хранилища запечатанного состояния часов.
type ClockStore interface {
	Load() ([]byte, error)
	Save(data []byte) error
}

// FileClockStore — файловое хранилище (dev). В проде заменить на TPM/keystore.
type FileClockStore struct{ Path string }

func (f FileClockStore) Load() ([]byte, error) {
	b, err := os.ReadFile(f.Path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return b, err
}

func (f FileClockStore) Save(data []byte) error {
	return os.WriteFile(f.Path, data, 0o600)
}

// NewSealedClock создаёт часы с install-secret и хранилищем. skew по умолчанию 5 мин.
func NewSealedClock(secret []byte, store ClockStore) *SealedClock {
	return &SealedClock{secret: secret, store: store, skew: 5 * time.Minute}
}

type sealedState struct {
	HighWater int64  `json:"hw"`
	MAC       string `json:"mac"`
}

func (c *SealedClock) mac(hw int64) string {
	m := hmac.New(sha256.New, c.secret)
	fmt.Fprintf(m, "%d", hw)
	return hex.EncodeToString(m.Sum(nil))
}

// load возвращает high-water и признак подделки (tampered).
func (c *SealedClock) load() (hw int64, tampered bool, err error) {
	data, err := c.store.Load()
	if err != nil {
		return 0, false, err
	}
	if len(data) == 0 {
		return 0, false, nil
	}
	var s sealedState
	if err := json.Unmarshal(data, &s); err != nil {
		return 0, true, nil // повреждённое состояние трактуем как подделку
	}
	if !hmac.Equal([]byte(s.MAC), []byte(c.mac(s.HighWater))) {
		return 0, true, nil
	}
	return s.HighWater, false, nil
}

func (c *SealedClock) save(hw int64) error {
	data, err := json.Marshal(sealedState{HighWater: hw, MAC: c.mac(hw)})
	if err != nil {
		return err
	}
	return c.store.Save(data)
}

// Observe двигает high-water вперёд по доверенному источнику времени
// (дата выпуска лицензии, подписанные бандлы обновлений, потоковая телеметрия).
func (c *SealedClock) Observe(t time.Time) error {
	hw, tampered, err := c.load()
	if err != nil {
		return err
	}
	if tampered {
		hw = 0 // переинициализируем от наблюдаемого времени
	}
	if t.Unix() > hw {
		return c.save(t.Unix())
	}
	return nil
}

// EffectiveNow возвращает эффективное (монотонное) время и признак отката/подделки.
func (c *SealedClock) EffectiveNow(localNow time.Time) (eff time.Time, rollback bool, err error) {
	hw, tampered, err := c.load()
	if err != nil {
		return localNow, false, err
	}
	rollback = tampered
	hwTime := time.Unix(hw, 0)

	// Локальные часы оказались раньше high-water (с поправкой на skew) => откат.
	if localNow.Add(c.skew).Before(hwTime) {
		rollback = true
	}

	eff = localNow
	if hwTime.After(eff) {
		eff = hwTime
	}
	// Продвигаем high-water до эффективного времени.
	if eff.Unix() > hw {
		_ = c.save(eff.Unix())
	}
	return eff, rollback, nil
}
