package mail

import (
	"testing"
	"time"
)

func TestValidateTokenOK(t *testing.T) {
	secret := []byte("test-secret")
	exp := time.Now().Add(time.Hour)
	raw, err := SignTestToken(secret, "t-demo", "alice@mail.gov.az", "u-alice", exp)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := ValidateToken(raw, secret)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if claims.TenantID != "t-demo" || claims.Email != "alice@mail.gov.az" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestValidateTokenExpired(t *testing.T) {
	secret := []byte("test-secret")
	raw, err := SignTestToken(secret, "t-demo", "alice@mail.gov.az", "u-alice", time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateToken(raw, secret); err == nil {
		t.Fatal("expected expired error")
	}
}

func TestValidateTokenWrongSecret(t *testing.T) {
	raw, err := SignTestToken([]byte("a"), "t-demo", "alice@mail.gov.az", "u-alice", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateToken(raw, []byte("b")); err == nil {
		t.Fatal("expected signature error")
	}
}

func TestValidateTokenMissingClaims(t *testing.T) {
	secret := []byte("test-secret")
	raw, err := SignTestToken(secret, "", "alice@mail.gov.az", "u-alice", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateToken(raw, secret); err == nil {
		t.Fatal("expected missing tenant_id error")
	}
}
