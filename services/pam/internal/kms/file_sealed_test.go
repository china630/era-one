package kms

import (
	"path/filepath"
	"testing"
)

func TestFileSealedSealUnseal(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "master.sealed")
	f, err := NewFileSealed(path)
	if err != nil {
		t.Fatal(err)
	}
	master := make([]byte, 32)
	for i := range master {
		master[i] = byte(i)
	}
	if err := f.SetMasterKey(master); err != nil {
		t.Fatal(err)
	}
	f.Clear()
	f2, err := NewFileSealed(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := f2.MasterKey()
	if err != nil {
		t.Fatal(err)
	}
	for i := range master {
		if got[i] != master[i] {
			t.Fatal("master key mismatch after file unseal")
		}
	}
	if f2.Name() != "file-sealed" {
		t.Fatal("name")
	}
}

func TestSelectProvider(t *testing.T) {
	p, err := SelectProvider("file-sealed", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if p.Name() != "file-sealed" {
		t.Fatalf("got %s", p.Name())
	}
}
