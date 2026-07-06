package api

import (
	"testing"

	"era/services/update-service/internal/bundle"
)

// M-08: bundle latest -> verify signed token (OTA mirror path).
func TestOTABundleVerifyE2E(t *testing.T) {
	priv, pub, err := bundle.LoadSigningKey()
	if err != nil {
		t.Fatal(err)
	}
	srv, err := New(priv, pub)
	if err != nil {
		t.Fatal(err)
	}
	if err := srv.refreshBundle(); err != nil {
		t.Fatal(err)
	}
	srv.mu.RLock()
	token := srv.token
	srv.mu.RUnlock()
	if token == "" {
		t.Fatal("empty bundle token")
	}
	claims, err := bundle.Verify(token, pub)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if claims.Kind == "" {
		t.Fatal("missing kind in claims")
	}
}
