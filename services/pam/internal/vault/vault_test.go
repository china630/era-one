package vault

import (
	"testing"

	"era/services/pam/internal/kms"
	"era/services/pam/internal/shamir"
)

func TestVaultEncryptAtRestGolden(t *testing.T) {
	master := make([]byte, 32)
	for i := range master {
		master[i] = byte(i + 1)
	}
	v := New(kms.NewSoftwareSealed())
	if err := v.Unseal(master); err != nil {
		t.Fatal(err)
	}
	meta, err := v.PutStatic("t1", "db-admin", "db01", "root", "SuperSecret!Pass")
	if err != nil {
		t.Fatal(err)
	}
	list := v.ListMeta()
	if len(list) != 1 || list[0].Name != "db-admin" {
		t.Fatal("meta list")
	}
	_, pass, err := v.Reveal(meta.ID)
	if err != nil || pass != "SuperSecret!Pass" {
		t.Fatalf("reveal: %v pass=%q", err, pass)
	}
	v.Seal()
	if _, _, err := v.Reveal(meta.ID); err == nil {
		t.Fatal("expected sealed error")
	}
}

func TestShamirUnsealFlow(t *testing.T) {
	master := []byte("01234567890123456789012345678901")
	shares, err := shamir.Split(master, 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	recovered, err := shamir.Combine(shares[:2])
	if err != nil {
		t.Fatal(err)
	}
	v := New(kms.NewSoftwareSealed())
	if err := v.Unseal(recovered); err != nil {
		t.Fatal(err)
	}
	if v.Sealed() {
		t.Fatal("should be unsealed")
	}
}
