package license

import (
	"crypto/ed25519"
	"testing"
	"time"
)

func sampleLease(now time.Time) *LeaseClaims {
	def := DefaultLeasePolicy()
	return &LeaseClaims{
		LicenseID:            "lic-golden-001",
		DeploymentID:         "deploy-az-001",
		TenantID:             "tenant-bank",
		Modules:              []Module{ModuleControlAI, ModuleResponse},
		IssuedAt:             now.Unix(),
		ExpiresAt:            now.AddDate(0, 0, 30).Unix(),
		GraceDays:            def.GraceDays,
		OfflineMaxDays:       def.OfflineMaxDays,
		RenewalIntervalHours: def.RenewalIntervalHours,
		DegradationMode:      def.DegradationMode,
	}
}

func TestLeaseSignVerifyRoundTrip(t *testing.T) {
	pub, priv, err := GenerateKeypair()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	token, err := SignLease(sampleLease(now), priv)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	got, err := VerifyLease(token, pub)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if got.LicenseID != "lic-golden-001" || !containsModule(got.Modules, ModuleControlAI) {
		t.Fatalf("claims: %+v", got)
	}
}

func containsModule(mods []Module, want Module) bool {
	for _, m := range mods {
		if m == want {
			return true
		}
	}
	return false
}

func TestLeaseEvaluateLifecycle(t *testing.T) {
	now := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	c := sampleLease(now)

	if ev := c.EvaluateLease(now.AddDate(0, 0, 10), "lic-golden-001", "deploy-az-001", now); ev.Status != LeaseStatusValid {
		t.Fatalf("ожидался VALID, получено %s", ev.Status)
	}
	if ev := c.EvaluateLease(now.AddDate(0, 0, 35), "lic-golden-001", "deploy-az-001", now); ev.Status != LeaseStatusGrace {
		t.Fatalf("ожидался GRACE, получено %s", ev.Status)
	}
	evExp := c.EvaluateLease(now.AddDate(0, 0, 70), "lic-golden-001", "deploy-az-001", now)
	if evExp.Status != LeaseStatusExpired || !evExp.Degraded {
		t.Fatalf("ожидался EXPIRED+degraded, получено %s degraded=%v", evExp.Status, evExp.Degraded)
	}
	if evExp.AllowNewOnboarding || evExp.AllowUpdates {
		t.Fatal("после exp+grace ожидалась деградация no_new_nodes и no_updates")
	}
}

func TestLeaseOfflineMaxDegrades(t *testing.T) {
	now := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	c := sampleLease(now)
	c.OfflineMaxDays = 30
	lastRenew := now.AddDate(0, 0, -31)

	ev := c.EvaluateLease(now, "lic-golden-001", "deploy-az-001", lastRenew)
	if !ev.Degraded || ev.AllowNewOnboarding || ev.AllowUpdates {
		t.Fatalf("offline max: %+v", ev)
	}
}

func TestLeaseLicenseMismatch(t *testing.T) {
	now := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	c := sampleLease(now)
	ev := c.EvaluateLease(now, "lic-other", "deploy-az-001", now)
	if ev.Status != LeaseStatusMismatch {
		t.Fatalf("ожидался MISMATCH, получено %s", ev.Status)
	}
}

func TestCheckLeaseIntegration(t *testing.T) {
	pub, priv, _ := GenerateKeypair()
	now := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	token, _ := SignLease(sampleLease(now), priv)
	ev, claims, err := CheckLease(token, pub, "lic-golden-001", "deploy-az-001", now, now)
	if err != nil || claims == nil || ev.Status != LeaseStatusValid {
		t.Fatalf("check: err=%v ev=%+v", err, ev)
	}
}

// goldenLeaseKey — детерминированный ключ только для golden-теста wire-формата.
func goldenLeaseKey() (ed25519.PublicKey, ed25519.PrivateKey) {
	seed := make([]byte, ed25519.SeedSize)
	copy(seed, []byte("era-lease-golden-test-seed-32b!"))
	priv := ed25519.NewKeyFromSeed(seed)
	return priv.Public().(ed25519.PublicKey), priv
}

func TestValidatorCheckWithLeaseAirGap(t *testing.T) {
	pub, priv, _ := GenerateKeypair()
	now := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	lic, _ := Sign(newClaims(now, 1), priv)
	v := &Validator{Pub: pub}
	ev, _, lev, lclaims, err := v.CheckWithLease(lic, "", now, time.Time{}, 1)
	if err != nil || lev.Status != LeaseStatusValid || lclaims != nil {
		t.Fatalf("air-gap: err=%v lev=%+v lclaims=%v", err, lev, lclaims)
	}
	if ev.Status != StatusValid {
		t.Fatalf("license: %s", ev.Status)
	}
}

func TestLeaseGoldenWireFormat(t *testing.T) {
	pub, priv := goldenLeaseKey()
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	claims := sampleLease(now)
	claims.LicenseID = "lic-golden-fixed"
	claims.DeploymentID = "deploy-fixed"
	claims.TenantID = "tenant-fixed"
	claims.IssuedAt = 1782916800
	claims.ExpiresAt = 1785508800

	token, err := SignLease(claims, priv)
	if err != nil {
		t.Fatal(err)
	}
	got, err := VerifyLease(token, pub)
	if err != nil {
		t.Fatal(err)
	}
	if got.LicenseID != "lic-golden-fixed" {
		t.Fatalf("roundtrip: %+v", got)
	}
	if token[:5] != LeaseFormat {
		t.Fatalf("bad prefix: %q", token[:5])
	}
}
