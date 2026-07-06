package shamir

import (
	"bytes"
	"testing"
)

func TestSplitCombineGolden(t *testing.T) {
	secret := []byte("era-pam-master-key-32-bytes!!!!!")
	shares, err := Split(secret, 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Combine([][]byte{shares[0], shares[2]})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(secret, got) {
		t.Fatalf("combine mismatch: %q vs %q", got, secret)
	}
}

func TestEncodeDecodeRoundtrip(t *testing.T) {
	shares, err := Split([]byte("test-key"), 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	enc := EncodeShares(shares)
	dec, err := DecodeShares(enc[:2])
	if err != nil {
		t.Fatal(err)
	}
	got, err := Combine(dec)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "test-key" {
		t.Fatalf("got %q", got)
	}
}
