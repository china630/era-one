package licensegate

import (
	"testing"
	"time"

	lic "era/services/license/pkg/license"
)

func TestGateFromEnvDevDefaultNoToken(t *testing.T) {
	t.Setenv("ERA_LICENSE_STRICT", "")
	t.Setenv("ERA_PRODUCTION", "")
	t.Setenv("ERA_ENV_PRODUCTION", "")
	t.Setenv("ERA_ENV", "")
	t.Setenv("ERA_LICENSE_TOKEN", "")
	t.Setenv("ERA_LICENSE_DEV", "")
	g, err := GateFromEnv(1)
	if err != nil {
		t.Fatal(err)
	}
	if !g.Allow(ModuleControlAI) {
		t.Fatal("expected dev default ai on")
	}
	if !g.Allow(ModulePerimeter) {
		t.Fatal("expected perimeter in DevDefault")
	}
}

func TestGateFromEnvLicenseDevAll(t *testing.T) {
	t.Setenv("ERA_LICENSE_STRICT", "")
	t.Setenv("ERA_PRODUCTION", "")
	t.Setenv("ERA_ENV_PRODUCTION", "")
	t.Setenv("ERA_ENV", "")
	t.Setenv("ERA_LICENSE_DEV", "1")
	g, err := GateFromEnv(0)
	if err != nil {
		t.Fatal(err)
	}
	if !g.Allow(ModuleFederated) {
		t.Fatal("ERA_LICENSE_DEV should enable all")
	}
}

func TestStrictModeEnvSync(t *testing.T) {
	clear := func() {
		t.Setenv("ERA_LICENSE_STRICT", "")
		t.Setenv("ERA_PRODUCTION", "")
		t.Setenv("ERA_ENV_PRODUCTION", "")
		t.Setenv("ERA_ENV", "")
	}
	clear()
	if StrictMode() {
		t.Fatal("default must not be strict")
	}
	t.Setenv("ERA_ENV_PRODUCTION", "1")
	if !StrictMode() {
		t.Fatal("ERA_ENV_PRODUCTION=1 must enable StrictMode")
	}
	clear()
	t.Setenv("ERA_ENV_PRODUCTION", "true")
	if !StrictMode() {
		t.Fatal("ERA_ENV_PRODUCTION=true must enable StrictMode")
	}
	clear()
	t.Setenv("ERA_ENV", "production")
	if !StrictMode() {
		t.Fatal("ERA_ENV=production must enable StrictMode")
	}
	clear()
	t.Setenv("ERA_ENV", "Production")
	if !StrictMode() {
		t.Fatal("ERA_ENV=Production (case-insensitive) must enable StrictMode")
	}
	clear()
	t.Setenv("ERA_PRODUCTION", "1")
	if !StrictMode() {
		t.Fatal("ERA_PRODUCTION=1 must enable StrictMode")
	}
	clear()
	t.Setenv("ERA_LICENSE_STRICT", "yes")
	if !StrictMode() {
		t.Fatal("ERA_LICENSE_STRICT=yes must enable StrictMode")
	}
}

func TestGateFromEnvStrictRequiresToken(t *testing.T) {
	t.Setenv("ERA_LICENSE_STRICT", "1")
	t.Setenv("ERA_LICENSE_TOKEN", "")
	t.Setenv("ERA_LICENSE_PATH", "")
	_, err := GateFromEnv(1)
	if err == nil {
		t.Fatal("expected error in strict mode without token")
	}
}

func TestGateFromEnvValidToken(t *testing.T) {
	pub, priv, err := lic.GenerateKeypair()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	c := &lic.Claims{
		Version: 1, LicenseID: "lic-test", Customer: "test",
		TenantID: "t1", Edition: "core",
		Modules: []lic.Module{lic.Module("control-ai"), lic.Module("manage")},
		MaxNodes: 100, IssuedAt: now.Unix(), NotBefore: now.Unix(),
		ExpiresAt: now.AddDate(1, 0, 0).Unix(), GraceDays: 7,
	}
	token, err := lic.Sign(c, priv)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("ERA_LICENSE_TOKEN", token)
	t.Setenv("ERA_VENDOR_PUB", lic.EncodeKey(pub))
	g, err := GateFromEnv(1)
	if err != nil {
		t.Fatal(err)
	}
	if !g.Allow(ModuleControlAI) || !g.Allow(ModuleManage) {
		t.Fatal("expected modules from token")
	}
	if g.Allow(ModuleResponse) {
		t.Fatal("response not in token")
	}
}

func TestValidateStartupExpiredStrict(t *testing.T) {
	pub, priv, _ := lic.GenerateKeypair()
	now := time.Now().UTC()
	c := &lic.Claims{
		Version: 1, LicenseID: "lic-exp", Customer: "test",
		TenantID: "t1", Edition: "core", MaxNodes: 10,
		IssuedAt: now.AddDate(-2, 0, 0).Unix(),
		NotBefore: now.AddDate(-2, 0, 0).Unix(),
		ExpiresAt: now.AddDate(-1, 0, 0).Unix(),
		GraceDays: 0,
	}
	token, _ := lic.Sign(c, priv)
	t.Setenv("ERA_LICENSE_STRICT", "1")
	t.Setenv("ERA_LICENSE_TOKEN", token)
	t.Setenv("ERA_VENDOR_PUB", lic.EncodeKey(pub))
	if err := ValidateStartup(1); err == nil {
		t.Fatal("expected expired license to fail startup")
	}
}
