package vault

import (
	"os"
	"path/filepath"
	"testing"

	"era/services/pam/internal/kms"
)

func TestVaultRestartPersistence(t *testing.T) {
	dir := t.TempDir()
	master := make([]byte, 32)
	for i := range master {
		master[i] = byte(i + 3)
	}

	ps, err := NewPersistStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	v1 := New(kms.NewSoftwareSealed())
	if err := v1.BindPersist(ps); err != nil {
		t.Fatal(err)
	}
	if err := v1.Unseal(master); err != nil {
		t.Fatal(err)
	}
	meta, err := v1.PutStatic("t1", "db", "db01", "root", "persist-secret")
	if err != nil {
		t.Fatal(err)
	}
	v1.Seal()

	// simulate process restart
	v2 := New(kms.NewSoftwareSealed())
	ps2, err := NewPersistStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := v2.BindPersist(ps2); err != nil {
		t.Fatal(err)
	}
	list := v2.ListMeta()
	if len(list) != 1 || list[0].ID != meta.ID {
		t.Fatalf("expected restored meta, got %+v", list)
	}
	if err := v2.Unseal(master); err != nil {
		t.Fatal(err)
	}
	_, pass, err := v2.Reveal(meta.ID)
	if err != nil || pass != "persist-secret" {
		t.Fatalf("reveal after restart: %v pass=%q", err, pass)
	}
	if _, err := os.Stat(filepath.Join(dir, vaultBlobName)); err != nil {
		t.Fatal("blob file missing")
	}
}
