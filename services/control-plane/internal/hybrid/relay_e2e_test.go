package hybrid

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	lic "era/services/license/pkg/license"
	"era/services/control-plane/internal/store"
)

func TestRelayE2EWithMockVendor(t *testing.T) {
	pub, priv, err := lic.GenerateKeypair()
	if err != nil {
		t.Fatal(err)
	}

	portal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/hybrid/lease/renew":
			def := lic.DefaultLeasePolicy()
			now := time.Now().UTC()
			lc := &lic.LeaseClaims{
				LicenseID: "lic-e2e", DeploymentID: "dep-e2e", TenantID: "t1",
				IssuedAt: now.Unix(), ExpiresAt: now.AddDate(0, 0, 30).Unix(),
				GraceDays: def.GraceDays, OfflineMaxDays: def.OfflineMaxDays,
				RenewalIntervalHours: def.RenewalIntervalHours,
				DegradationMode:      def.DegradationMode,
			}
			token, _ := lic.SignLease(lc, priv)
			_ = json.NewEncoder(w).Encode(map[string]string{"token": token})
		case "/api/v1/hybrid/crl":
			token, _ := lic.SignCRL(&lic.CRL{IssuedAt: time.Now().UTC().Unix(), Revoked: nil}, priv)
			_ = json.NewEncoder(w).Encode(map[string]string{"token": token})
		case "/api/v1/hybrid/health":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer portal.Close()

	btoken := signTestBundle(t, priv, "bnd-e2e", "sigma-corpus")
	update := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"token": btoken})
	}))
	defer update.Close()

	pu, _ := url.Parse(portal.URL)
	uu, _ := url.Parse(update.URL)

	st := store.NewMemory()
	st.SetHybridPolicy(store.HybridPolicy{
		Enabled: true, PortalURL: portal.URL, UpdateURL: update.URL,
		DeploymentID: "dep-e2e", LicenseID: "lic-e2e",
		EgressAllowlist: []string{pu.Hostname(), uu.Hostname()},
		HealthLevel:     "A",
	})
	t.Setenv("ERA_VENDOR_PUB", lic.EncodeKey(pub))
	t.Setenv("ERA_TLS_INSECURE", "1")

	relay, err := NewRelay(st)
	if err != nil {
		t.Fatal(err)
	}
	relay.SyncOnce()

	rt := st.GetHybridRuntime()
	if rt.LeaseStatus == "" {
		t.Fatalf("lease status empty, err=%q", rt.LastError)
	}
	if rt.LastBundleID == "" {
		t.Fatal("bundle not applied")
	}
	pol := st.Policy()
	if pol.Version == "3.0.0-ga" {
		t.Fatalf("expected policy version bump after sigma bundle, got %q", pol.Version)
	}
	if pol.Rules["sigma"] == "" {
		t.Fatal("sigma rules ref not updated")
	}
	if len(st.ListEgressAudit(10)) < 2 {
		t.Fatal("expected egress audit")
	}
}

func signTestBundle(t *testing.T, priv ed25519.PrivateKey, id, kind string) string {
	t.Helper()
	payload, _ := json.Marshal(map[string]string{"id": id, "kind": kind})
	sig := ed25519.Sign(priv, payload)
	return strings.Join([]string{
		bundleFormat,
		base64.RawURLEncoding.EncodeToString(payload),
		base64.RawURLEncoding.EncodeToString(sig),
	}, ".")
}

func TestRelayAirGapDefault(t *testing.T) {
	st := store.NewMemory()
	relay, err := NewRelay(st)
	if err != nil {
		t.Fatal(err)
	}
	relay.SyncOnce()
	if st.GetHybridPolicy().Enabled {
		t.Fatal("connected must default OFF")
	}
}
