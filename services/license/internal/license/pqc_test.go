package license

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"
)

func TestPQCReadiness(t *testing.T) {
	if !SupportsPQC() {
		t.Fatal("PQC should be supported")
	}
	if PreferredSignAlgorithm() != SigEd25519 {
		t.Fatal("production default remains ed25519")
	}
	if VerifyAlgorithmForToken(FormatPQC) != SigHybridEd25519MLDSA {
		t.Fatal("PQC token routing")
	}
}

func TestHybridSignVerifyRoundTrip(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	c := &Claims{
		Version: 1, LicenseID: "lic-pqc-1", Customer: "Bank", TenantID: "t1",
		Edition: "core", Modules: []Module{ModuleControlAI}, MaxNodes: 100,
		IssuedAt: now.Unix(), NotBefore: now.Unix(), ExpiresAt: now.Add(365 * 24 * time.Hour).Unix(),
	}
	token, err := SignHybrid(c, priv)
	if err != nil {
		t.Fatal(err)
	}
	got, err := VerifyHybrid(token, pub)
	if err != nil {
		t.Fatal(err)
	}
	if got.LicenseID != c.LicenseID {
		t.Fatalf("lid=%s", got.LicenseID)
	}
}

func TestVerifyAnyRoutesPQC(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	c := &Claims{Version: 1, LicenseID: "x", IssuedAt: 1, NotBefore: 1, ExpiresAt: 9999999999}
	token, err := SignHybrid(c, priv)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyAny(token, pub); err != nil {
		t.Fatal(err)
	}
}

func TestHybridRejectsTamperedPayload(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	c := &Claims{Version: 1, LicenseID: "x", IssuedAt: 1, NotBefore: 1, ExpiresAt: 9999999999}
	token, _ := SignHybrid(c, priv)
	bad := token[:len(token)-2] + "AA"
	if _, err := VerifyHybrid(bad, pub); err == nil {
		t.Fatal("expected verify failure")
	}
}
