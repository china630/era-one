package licensegate

import (
	"os"
	"testing"
	"time"

	lic "era/services/license/pkg/license"
)

func TestGateFromEnvDevDefaultNoToken(t *testing.T) {
	os.Unsetenv("ERA_LICENSE_STRICT")
	os.Unsetenv("ERA_PRODUCTION")
	os.Unsetenv("ERA_LICENSE_TOKEN")
	g, err := GateFromEnv(1)
	if err != nil {
		t.Fatal(err)
	}
	if !g.Allow(ModuleControlAI) {
		t.Fatal("expected dev default ai on")
	}
}

func TestGateFromEnvStrictRequiresToken(t *testing.T) {
	os.Setenv("ERA_LICENSE_STRICT", "1")
	defer os.Unsetenv("ERA_LICENSE_STRICT")
	os.Unsetenv("ERA_LICENSE_TOKEN")
	os.Unsetenv("ERA_LICENSE_PATH")
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
	os.Setenv("ERA_LICENSE_TOKEN", token)
	os.Setenv("ERA_VENDOR_PUB", lic.EncodeKey(pub))
	defer func() {
		os.Unsetenv("ERA_LICENSE_TOKEN")
		os.Unsetenv("ERA_VENDOR_PUB")
	}()
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
	os.Setenv("ERA_LICENSE_STRICT", "1")
	os.Setenv("ERA_LICENSE_TOKEN", token)
	os.Setenv("ERA_VENDOR_PUB", lic.EncodeKey(pub))
	defer func() {
		os.Unsetenv("ERA_LICENSE_STRICT")
		os.Unsetenv("ERA_LICENSE_TOKEN")
		os.Unsetenv("ERA_VENDOR_PUB")
	}()
	if err := ValidateStartup(1); err == nil {
		t.Fatal("expected expired license to fail startup")
	}
}
