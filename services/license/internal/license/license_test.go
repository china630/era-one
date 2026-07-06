package license

import (
	"testing"
	"time"
)

func newClaims(now time.Time, years int) *Claims {
	return &Claims{
		LicenseID: "lic-001",
		Customer:  "Test Bank",
		TenantID:  "tenant-1",
		Edition:   "core",
		Modules:   []Module{ModuleVuln, ModuleControlAI},
		MaxNodes:  1000,
		IssuedAt:  now.Unix(),
		NotBefore: now.Unix(),
		ExpiresAt: now.AddDate(years, 0, 0).Unix(),
		GraceDays: 30,
	}
}

func TestHasModuleLegacyAI(t *testing.T) {
	c := &Claims{Modules: []Module{ModuleAILegacy}}
	if !c.HasModule(ModuleControlAI) {
		t.Fatal("legacy ai token must satisfy HasModule(control-ai)")
	}
}

func TestSignVerifyRoundTrip(t *testing.T) {
	pub, priv, err := GenerateKeypair()
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}
	now := time.Now()
	token, err := Sign(newClaims(now, 1), priv)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	got, err := Verify(token, pub)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if got.Customer != "Test Bank" || !got.HasModule(ModuleControlAI) {
		t.Fatalf("claims не совпадают: %+v", got)
	}
}

func TestVerifyRejectsWrongKey(t *testing.T) {
	_, priv, _ := GenerateKeypair()
	otherPub, _, _ := GenerateKeypair()
	token, _ := Sign(newClaims(time.Now(), 1), priv)
	if _, err := Verify(token, otherPub); err == nil {
		t.Fatal("ожидалась ошибка подписи для чужого ключа")
	}
}

func TestVerifyRejectsTamper(t *testing.T) {
	pub, priv, _ := GenerateKeypair()
	token, _ := Sign(newClaims(time.Now(), 1), priv)
	tampered := token[:len(token)-2] + "AA"
	if _, err := Verify(tampered, pub); err == nil {
		t.Fatal("ожидалась ошибка для изменённого токена")
	}
}

func TestEvaluateLifecycle(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	c := newClaims(now, 1) // exp = 2027-01-01, grace 30 дней

	if ev := c.Evaluate(now.AddDate(0, 1, 0), "", 10); ev.Status != StatusValid {
		t.Fatalf("ожидался VALID, получено %s", ev.Status)
	}
	if ev := c.Evaluate(now.AddDate(0, -1, 0), "", 10); ev.Status != StatusNotYetValid {
		t.Fatalf("ожидался NOT_YET_VALID, получено %s", ev.Status)
	}
	// через 1 год + 10 дней -> grace
	if ev := c.Evaluate(now.AddDate(1, 0, 10), "", 10); ev.Status != StatusGrace {
		t.Fatalf("ожидался GRACE, получено %s", ev.Status)
	}
	// через 1 год + 40 дней -> expired (деградация)
	if ev := c.Evaluate(now.AddDate(1, 0, 40), "", 10); ev.Status != StatusExpired || !ev.Degraded {
		t.Fatalf("ожидался EXPIRED+degraded, получено %s", ev.Status)
	}
	// превышение лимита узлов
	if ev := c.Evaluate(now.AddDate(0, 1, 0), "", 5000); ev.Status != StatusNodeLimitExceeded {
		t.Fatalf("ожидался NODE_LIMIT_EXCEEDED, получено %s", ev.Status)
	}
}

func TestDeploymentBinding(t *testing.T) {
	now := time.Now()
	c := newClaims(now, 1)
	c.Deployment = "deploy-AAA"
	if ev := c.Evaluate(now, "deploy-BBB", 1); ev.Status != StatusExpired {
		t.Fatalf("ожидалась блокировка по привязке, получено %s", ev.Status)
	}
	if ev := c.Evaluate(now, "deploy-AAA", 1); ev.Status != StatusValid {
		t.Fatalf("ожидался VALID для своего развёртывания, получено %s", ev.Status)
	}
}
