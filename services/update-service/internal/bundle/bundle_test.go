package bundle

import (
	"crypto/ed25519"
	"testing"
	"time"
)

func TestBundleSignVerifyRoundTrip(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	c := &Claims{
		Version:      ClaimsVersion,
		BundleID:     "bnd-golden-001",
		Kind:         "sigma-corpus",
		IssuedAt:     now.Unix(),
		FileCount:    2,
		ManifestHash: "abc123deadbeef",
	}
	token, err := SignClaims(c, priv)
	if err != nil {
		t.Fatal(err)
	}
	if token[:8] != "ERABNDL1" {
		t.Fatalf("bad prefix: %s", token[:8])
	}
	got, err := Verify(token, pub)
	if err != nil {
		t.Fatal(err)
	}
	if got.BundleID != c.BundleID || got.Kind != c.Kind {
		t.Fatalf("claims: %+v", got)
	}
}

func TestBundleWireFormatStable(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	c := &Claims{
		Version:      ClaimsVersion,
		BundleID:     "bnd-wire",
		Kind:         "sigma-corpus",
		IssuedAt:     1750000000,
		FileCount:    1,
		ManifestHash: "deadbeef",
	}
	t1, _ := SignClaims(c, priv)
	t2, _ := SignClaims(c, priv)
	if t1 != t2 {
		t.Fatal("wire format must be deterministic for same claims")
	}
	if _, err := Verify(t1, pub); err != nil {
		t.Fatal(err)
	}
}

func TestHashManifestStable(t *testing.T) {
	m := &Manifest{Files: []FileEntry{{Path: "a.yml", SHA256: "dead", Size: 10}}}
	h1 := HashManifest(m)
	h2 := HashManifest(m)
	if h1 != h2 || h1 == "" {
		t.Fatalf("hash unstable: %s %s", h1, h2)
	}
}
