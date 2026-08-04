package autodiscover

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"era/services/platform/tenant"
)

func TestRenderGoldenTLS(t *testing.T) {
	tenants := tenant.NewStore()
	_ = tenants.PutTenant(tenant.Tenant{ID: "t1", Name: "Gov", Slug: "gov"})
	_ = tenants.PutDomain(tenant.Domain{ID: "d1", TenantID: "t1", FQDN: "mail.gov.az", Primary: true})

	got, err := Render(Config{
		Email:      "alice@mail.gov.az",
		Tenants:    tenants,
		MailHost:   "mail.mail.gov.az",
		IMAPHost:   "mail.mail.gov.az",
		SMTPHost:   "mail.mail.gov.az",
		EWSHost:    "mail.mail.gov.az",
		CalDAVHost: "mail.mail.gov.az",
		IMAPPort:   993,
		SMTPPort:   587,
		HTTPPort:   8150,
		UseTLS:     true,
		SMTPUseTLS: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	goldenPath := filepath.Join("testdata", "autodiscover_alice_tls.golden.xml")
	if os.Getenv("ERA_UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Log("golden updated")
		return
	}

	wantBytes, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v (run with ERA_UPDATE_GOLDEN=1)", err)
	}
	want := strings.ReplaceAll(string(wantBytes), "\r\n", "\n")
	gotNorm := strings.ReplaceAll(got, "\r\n", "\n")
	if gotNorm != want {
		t.Fatalf("autodiscover TLS mismatch; set ERA_UPDATE_GOLDEN=1 to update\n--- got ---\n%s\n--- want ---\n%s", gotNorm, want)
	}
}

func TestRenderGolden(t *testing.T) {
	tenants := tenant.NewStore()
	_ = tenants.PutTenant(tenant.Tenant{ID: "t1", Name: "Gov", Slug: "gov"})
	_ = tenants.PutDomain(tenant.Domain{ID: "d1", TenantID: "t1", FQDN: "mail.gov.az", Primary: true})

	got, err := Render(Config{
		Email:      "alice@mail.gov.az",
		Tenants:    tenants,
		MailHost:   "mail.mail.gov.az",
		IMAPHost:   "mail.mail.gov.az",
		SMTPHost:   "mail.mail.gov.az",
		EWSHost:    "mail.mail.gov.az",
		CalDAVHost: "mail.mail.gov.az",
		IMAPPort:   1143,
		SMTPPort:   2525,
		HTTPPort:   8150,
	})
	if err != nil {
		t.Fatal(err)
	}

	goldenPath := filepath.Join("testdata", "autodiscover_alice.golden.xml")
	if os.Getenv("ERA_UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Log("golden updated")
		return
	}

	wantBytes, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v (run with ERA_UPDATE_GOLDEN=1)", err)
	}
	want := strings.ReplaceAll(string(wantBytes), "\r\n", "\n")
	gotNorm := strings.ReplaceAll(got, "\r\n", "\n")
	if gotNorm != want {
		t.Fatalf("autodiscover mismatch; set ERA_UPDATE_GOLDEN=1 to update\n--- got ---\n%s\n--- want ---\n%s", gotNorm, want)
	}
}

func TestRenderUnknownDomain(t *testing.T) {
	tenants := tenant.NewStore()
	_, err := Render(Config{
		Email:   "bob@unknown.example",
		Tenants: tenants,
	})
	if err == nil {
		t.Fatal("expected error for unknown domain")
	}
}
