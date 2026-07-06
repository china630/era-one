// Package kms — абстракция мастер-ключа (HSM/KMS/TPM/dev software-sealed).
package kms

import "errors"

// Provider хранит мастер-ключ vault после unseal.
type Provider interface {
	Name() string
	SetMasterKey(key []byte) error
	MasterKey() ([]byte, error)
	Clear()
}

// SoftwareSealed — dev-провайдер (in-memory, ADR-0013 §6; prod → HSM gate).
type SoftwareSealed struct {
	key []byte
}

func NewSoftwareSealed() *SoftwareSealed {
	return &SoftwareSealed{}
}

func (s *SoftwareSealed) Name() string { return "software-sealed-dev" }

func (s *SoftwareSealed) SetMasterKey(key []byte) error {
	if len(key) != 32 {
		return errors.New("master key must be 32 bytes")
	}
	cp := make([]byte, 32)
	copy(cp, key)
	s.key = cp
	return nil
}

func (s *SoftwareSealed) MasterKey() ([]byte, error) {
	if len(s.key) == 0 {
		return nil, errors.New("kms: no master key")
	}
	cp := make([]byte, 32)
	copy(cp, s.key)
	return cp, nil
}

func (s *SoftwareSealed) Clear() {
	for i := range s.key {
		s.key[i] = 0
	}
	s.key = nil
}
