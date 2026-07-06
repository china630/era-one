package kms

import "errors"

// HSM — интерфейс аппаратного модуля (TPM/HSM); prod seal/unseal без RAM-only ключа.
type HSM interface {
	Provider
	// SealKey сохраняет мастер-ключ в защищённом хранилище HSM.
	SealKey(key []byte) error
	// UnsealKey извлекает мастер-ключ из HSM (требует PIN/attestation в prod).
	UnsealKey() ([]byte, error)
	// Available сообщает, подключён ли HSM.
	Available() bool
}

// ErrHSMAbsent — HSM не сконфигурирован.
var ErrHSMAbsent = errors.New("kms: hsm not available")

// StubHSM — заглушка для тестов и dev без железа.
type StubHSM struct {
	key []byte
	ok  bool
}

func NewStubHSM(available bool) *StubHSM {
	return &StubHSM{ok: available}
}

func (h *StubHSM) Name() string { return "hsm-stub" }

func (h *StubHSM) SetMasterKey(key []byte) error {
	if len(key) != 32 {
		return errors.New("master key must be 32 bytes")
	}
	cp := make([]byte, 32)
	copy(cp, key)
	h.key = cp
	return nil
}

func (h *StubHSM) MasterKey() ([]byte, error) {
	if len(h.key) == 0 {
		return nil, errors.New("kms: no master key")
	}
	cp := make([]byte, 32)
	copy(cp, h.key)
	return cp, nil
}

func (h *StubHSM) Clear() {
	for i := range h.key {
		h.key[i] = 0
	}
	h.key = nil
}

func (h *StubHSM) SealKey(key []byte) error {
	if !h.ok {
		return ErrHSMAbsent
	}
	return h.SetMasterKey(key)
}

func (h *StubHSM) UnsealKey() ([]byte, error) {
	if !h.ok {
		return nil, ErrHSMAbsent
	}
	return h.MasterKey()
}

func (h *StubHSM) Available() bool { return h.ok }
