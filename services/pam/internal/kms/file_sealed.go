package kms

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
)

type sealedBlob struct {
	Ciphertext []byte `json:"ciphertext"`
}

// FileSealed — prod-провайдер: мастер-ключ в RAM после unseal, at-rest seal в файле.
type FileSealed struct {
	path    string
	wrapKey []byte
	key     []byte
}

// NewFileSealed создаёт провайдер с seal-файлом по path.
func NewFileSealed(path string) (*FileSealed, error) {
	if path == "" {
		return nil, errors.New("file-sealed path required")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	f := &FileSealed{path: path}
	if err := f.loadOrCreateWrapKey(); err != nil {
		return nil, err
	}
	return f, nil
}

func (f *FileSealed) Name() string { return "file-sealed" }

func (f *FileSealed) loadOrCreateWrapKey() error {
	wrapPath := f.path + ".wrap"
	raw, err := os.ReadFile(wrapPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		k := make([]byte, 32)
		if _, err := rand.Read(k); err != nil {
			return err
		}
		if err := os.WriteFile(wrapPath, k, 0o600); err != nil {
			return err
		}
		f.wrapKey = k
		return nil
	}
	if len(raw) == 32 {
		f.wrapKey = append([]byte(nil), raw...)
		return nil
	}
	sum := sha256.Sum256(raw)
	f.wrapKey = sum[:]
	return nil
}

func (f *FileSealed) SetMasterKey(key []byte) error {
	if len(key) != 32 {
		return errors.New("master key must be 32 bytes")
	}
	cp := make([]byte, 32)
	copy(cp, key)
	f.key = cp
	return f.sealToFile(cp)
}

func (f *FileSealed) MasterKey() ([]byte, error) {
	if len(f.key) == 0 {
		if err := f.unsealFromFile(); err != nil {
			return nil, err
		}
	}
	if len(f.key) == 0 {
		return nil, errors.New("kms: no master key")
	}
	cp := make([]byte, 32)
	copy(cp, f.key)
	return cp, nil
}

func (f *FileSealed) Clear() {
	for i := range f.key {
		f.key[i] = 0
	}
	f.key = nil
}

func (f *FileSealed) sealToFile(key []byte) error {
	ct, err := encryptWrap(f.wrapKey, key)
	if err != nil {
		return err
	}
	b, _ := json.Marshal(sealedBlob{Ciphertext: ct})
	tmp := f.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, f.path)
}

func (f *FileSealed) unsealFromFile() error {
	raw, err := os.ReadFile(f.path)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.New("kms: sealed file missing")
		}
		return err
	}
	var blob sealedBlob
	if err := json.Unmarshal(raw, &blob); err != nil {
		return err
	}
	plain, err := decryptWrap(f.wrapKey, blob.Ciphertext)
	if err != nil {
		return err
	}
	if len(plain) != 32 {
		return errors.New("kms: invalid sealed key length")
	}
	f.key = plain
	return nil
}

func encryptWrap(key, plain []byte) ([]byte, error) {
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

func decryptWrap(key, ct []byte) ([]byte, error) {
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
		return nil, errors.New("kms: ciphertext too short")
	}
	return gcm.Open(nil, ct[:ns], ct[ns:], nil)
}

// SelectProvider выбирает KMS по ERA_KMS_PROVIDER.
func SelectProvider(name, stateDir string) (Provider, error) {
	switch name {
	case "", "software-sealed-dev":
		return NewSoftwareSealed(), nil
	case "file-sealed":
		path := filepath.Join(stateDir, "master.sealed")
		return NewFileSealed(path)
	default:
		return NewSoftwareSealed(), nil
	}
}
