package license

import (
	"testing"
	"time"
)

func TestCRLRoundTripAndRevocation(t *testing.T) {
	pub, priv, err := GenerateKeypair()
	if err != nil {
		t.Fatal(err)
	}
	crl := &CRL{
		IssuedAt: time.Now().Unix(),
		Revoked:  []string{"lic-aaa", "lic-bbb"},
	}
	token, err := SignCRL(crl, priv)
	if err != nil {
		t.Fatalf("sign crl: %v", err)
	}
	got, err := VerifyCRL(token, pub)
	if err != nil {
		t.Fatalf("verify crl: %v", err)
	}
	if !got.IsRevoked("lic-aaa") {
		t.Fatal("lic-aaa должна быть отозвана")
	}
	if got.IsRevoked("lic-ccc") {
		t.Fatal("lic-ccc не должна числиться отозванной")
	}
}

func TestCRLRejectsWrongKey(t *testing.T) {
	_, priv, _ := GenerateKeypair()
	otherPub, _, _ := GenerateKeypair()
	token, _ := SignCRL(&CRL{Revoked: []string{"lic-x"}}, priv)
	if _, err := VerifyCRL(token, otherPub); err == nil {
		t.Fatal("ожидалась ошибка подписи CRL для чужого ключа")
	}
}
