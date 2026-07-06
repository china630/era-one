package licensegate

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	lic "era/services/license/pkg/license"
)

// C-12: sealed clock детектирует rollback времени при проверке лицензии.
func TestSealedClockRollbackBlocksLicense(t *testing.T) {
	pub, priv, err := lic.GenerateKeypair()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	c := &lic.Claims{
		Version: 1, LicenseID: "lic-clock", Customer: "test",
		TenantID: "t1", Edition: "core", MaxNodes: 10,
		IssuedAt: now.Unix(), NotBefore: now.Unix(),
		ExpiresAt: now.AddDate(1, 0, 0).Unix(), GraceDays: 7,
	}
	token, err := lic.Sign(c, priv)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	clockPath := filepath.Join(dir, "sealed.clock")
	os.Setenv("ERA_LICENSE_TOKEN", token)
	os.Setenv("ERA_VENDOR_PUB", lic.EncodeKey(pub))
	os.Setenv("ERA_SEALED_CLOCK_PATH", clockPath)
	os.Setenv("ERA_SEALED_CLOCK_SECRET", "test-install-secret")
	defer func() {
		os.Unsetenv("ERA_LICENSE_TOKEN")
		os.Unsetenv("ERA_VENDOR_PUB")
		os.Unsetenv("ERA_SEALED_CLOCK_PATH")
		os.Unsetenv("ERA_SEALED_CLOCK_SECRET")
	}()
	if err := ValidateStartup(1); err != nil {
		t.Fatalf("first check: %v", err)
	}
	// Симулируем rollback: записываем будущее HW время в store.
	clock := lic.NewSealedClock([]byte("test-install-secret"), lic.FileClockStore{Path: clockPath})
	future := now.Add(48 * time.Hour)
	if err := clock.Observe(future); err != nil {
		t.Fatal(err)
	}
	if err := ValidateStartup(1); err == nil {
		t.Fatal("expected rollback/tamper to fail license check")
	}
}
