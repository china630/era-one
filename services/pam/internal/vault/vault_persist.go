package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
)

const vaultBlobName = "vault.blob"

type persistSnapshot struct {
	ShareHints []string                  `json:"share_hints"`
	Secrets    map[string]*storedSecret  `json:"secrets"`
}

// PersistStore — encrypted at-rest blob в ERA_PAM_STATE.
type PersistStore struct {
	mu   sync.Mutex
	dir  string
	key  []byte // storage seal key (32 bytes)
}

// NewPersistStore создаёт store в dir (обычно ERA_PAM_STATE).
func NewPersistStore(dir string) (*PersistStore, error) {
	if dir == "" {
		return nil, errors.New("persist dir required")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	ps := &PersistStore{dir: dir}
	if err := ps.loadOrCreateSealKey(); err != nil {
		return nil, err
	}
	return ps, nil
}

func (ps *PersistStore) blobPath() string {
	return filepath.Join(ps.dir, vaultBlobName)
}

func (ps *PersistStore) loadOrCreateSealKey() error {
	keyPath := filepath.Join(ps.dir, ".seal_key")
	raw, err := os.ReadFile(keyPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		k := make([]byte, 32)
		if _, err := rand.Read(k); err != nil {
			return err
		}
		if err := os.WriteFile(keyPath, k, 0o600); err != nil {
			return err
		}
		ps.key = k
		return nil
	}
	if len(raw) != 32 {
		sum := sha256.Sum256(raw)
		ps.key = sum[:]
		return nil
	}
	ps.key = append([]byte(nil), raw...)
	return nil
}

// Save шифрует snapshot и пишет на диск.
func (ps *PersistStore) Save(snap *persistSnapshot) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	plain, err := json.Marshal(snap)
	if err != nil {
		return err
	}
	ct, err := sealBlob(ps.key, plain)
	if err != nil {
		return err
	}
	tmp := ps.blobPath() + ".tmp"
	if err := os.WriteFile(tmp, ct, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, ps.blobPath())
}

// Load читает snapshot с диска (nil если файла нет).
func (ps *PersistStore) Load() (*persistSnapshot, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	raw, err := os.ReadFile(ps.blobPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	plain, err := openBlob(ps.key, raw)
	if err != nil {
		return nil, err
	}
	var snap persistSnapshot
	if err := json.Unmarshal(plain, &snap); err != nil {
		return nil, err
	}
	if snap.Secrets == nil {
		snap.Secrets = make(map[string]*storedSecret)
	}
	return &snap, nil
}

func sealBlob(key, plain []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plain, nil), nil
}

func openBlob(key, ct []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	ns := gcm.NonceSize()
	if len(ct) < ns {
		return nil, errors.New("blob too short")
	}
	return gcm.Open(nil, ct[:ns], ct[ns:], nil)
}

// BindPersist подключает durable store к vault.
func (v *Vault) BindPersist(ps *PersistStore) error {
	v.mu.Lock()
	v.persist = ps
	v.mu.Unlock()
	snap, err := ps.Load()
	if err != nil {
		return err
	}
	if snap == nil {
		return nil
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	v.shareHints = append([]string(nil), snap.ShareHints...)
	v.secrets = snap.Secrets
	return nil
}

func (v *Vault) flushPersist() {
	if v.persist == nil {
		return
	}
	snap := &persistSnapshot{
		ShareHints: append([]string(nil), v.shareHints...),
		Secrets:    v.secrets,
	}
	_ = v.persist.Save(snap)
}
