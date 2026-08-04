package autodiscover_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	bridgead "era/services/comms/mail-bridge/internal/autodiscover"
	"era/services/platform/tenant"
)

func TestAutodiscoverTenantGolden(t *testing.T) {
	ts := tenant.NewStore()
	_ = ts.PutTenant(tenant.Tenant{ID: "t-iw", Name: "IW", Slug: "iw"})
	_ = ts.PutDomain(tenant.Domain{ID: "d-iw", TenantID: "t-iw", FQDN: "mail.lab.local", Primary: true})
	_ = ts.PutTenant(tenant.Tenant{ID: "t-cg", Name: "CG", Slug: "cg"})
	_ = ts.PutDomain(tenant.Domain{ID: "d-cg", TenantID: "t-cg", FQDN: "cg.lab.local", Primary: true})

	cases := []struct {
		email  string
		golden string
	}{
		{"pilot@mail.lab.local", "autodiscover_icewarp_tenant.golden.xml"},
		{"pilot@cg.lab.local", "autodiscover_cg_tenant.golden.xml"},
	}
	for _, tc := range cases {
		got, err := bridgead.Render(bridgead.Config{
			Email:      tc.email,
			BridgeHost: "bridge.lab.local",
			HTTPPort:   8151,
			UseTLS:     true,
			Tenants:    ts,
		})
		if err != nil {
			t.Fatalf("%s: %v", tc.email, err)
		}
		wantBytes, err := os.ReadFile(filepath.Join("..", "testdata", tc.golden))
		if err != nil {
			t.Fatal(err)
		}
		normalize := func(s string) string {
			return strings.Join(strings.Fields(s), " ")
		}
		if normalize(got) != normalize(string(wantBytes)) {
			t.Fatalf("%s mismatch\ngot=%s\nwant=%s", tc.email, got, wantBytes)
		}
		if !strings.Contains(got, "/ews/Exchange.asmx") {
			t.Fatalf("EwsUrl must point to bridge")
		}
	}
}
