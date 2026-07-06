// Package vault — encrypted at-rest secret store (ADR-0013).
package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"era/services/pam/internal/kms"
	"github.com/google/uuid"
)

// SecretMeta — метаданные без plaintext (zero-knowledge list).
type SecretMeta struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Name      string    `json:"name"`
	Target    string    `json:"target"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}

type storedSecret struct {
	Meta       SecretMeta
	Ciphertext []byte
}

// Vault — seal/unseal + static secret engine.
type Vault struct {
	mu     sync.RWMutex
	kms    kms.Provider
	sealed bool
	// operator shares (hex) persisted for re-unseal in dev
	shareHints []string
	secrets    map[string]*storedSecret
	persist    *PersistStore
}

func New(k kms.Provider) *Vault {
	return &Vault{kms: k, sealed: true, secrets: make(map[string]*storedSecret)}
}

func (v *Vault) Sealed() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.sealed
}

func (v *Vault) Unseal(masterKey []byte) error {
	if len(masterKey) != 32 {
		return errors.New("master key must be 32 bytes")
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if err := v.kms.SetMasterKey(masterKey); err != nil {
		return err
	}
	v.sealed = false
	return nil
}

func (v *Vault) Seal() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.kms.Clear()
	v.sealed = true
}

func (v *Vault) SetShareHints(hints []string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.shareHints = append([]string(nil), hints...)
	v.flushPersist()
}

func (v *Vault) ShareHints() []string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return append([]string(nil), v.shareHints...)
}

func (v *Vault) PutStatic(tenantID, name, target, username, password string) (SecretMeta, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.sealed {
		return SecretMeta{}, errors.New("vault sealed")
	}
	ct, err := v.encrypt([]byte(password))
	if err != nil {
		return SecretMeta{}, err
	}
	meta := SecretMeta{
		ID: uuid.NewString(), TenantID: tenantID, Name: name,
		Target: target, Username: username, CreatedAt: time.Now().UTC(),
	}
	v.secrets[meta.ID] = &storedSecret{Meta: meta, Ciphertext: ct}
	v.flushPersist()
	return meta, nil
}

func (v *Vault) ListMeta() []SecretMeta {
	v.mu.RLock()
	defer v.mu.RUnlock()
	out := make([]SecretMeta, 0, len(v.secrets))
	for _, s := range v.secrets {
		out = append(out, s.Meta)
	}
	return out
}

func (v *Vault) GetMeta(id string) (SecretMeta, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	s, ok := v.secrets[id]
	if !ok {
		return SecretMeta{}, false
	}
	return s.Meta, true
}

func (v *Vault) Reveal(id string) (username, password string, err error) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if v.sealed {
		return "", "", errors.New("vault sealed")
	}
	s, ok := v.secrets[id]
	if !ok {
		return "", "", errors.New("not found")
	}
	plain, err := v.decrypt(s.Ciphertext)
	if err != nil {
		return "", "", err
	}
	return s.Meta.Username, string(plain), nil
}

// RotatePassword заменяет plaintext пароль секрета (ротация).
func (v *Vault) RotatePassword(id, newPassword string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.sealed {
		return errors.New("vault sealed")
	}
	s, ok := v.secrets[id]
	if !ok {
		return errors.New("not found")
	}
	ct, err := v.encrypt([]byte(newPassword))
	if err != nil {
		return err
	}
	s.Ciphertext = ct
	return nil
}

func (v *Vault) encrypt(plaintext []byte) ([]byte, error) {
	key, err := v.kms.MasterKey()
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func (v *Vault) decrypt(ciphertext []byte) ([]byte, error) {
	key, err := v.kms.MasterKey()
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}
	return gcm.Open(nil, ciphertext[:nonceSize], ciphertext[nonceSize:], nil)
}

// CustodyPayload для hash-chain аудита доступа к секрету.
func CustodyPayload(action, secretID, actor, tenantID string) []byte {
	b, _ := json.Marshal(map[string]string{
		"action": action, "secret_id": secretID, "actor": actor, "tenant_id": tenantID,
	})
	return b
}
