// ERA Mail Server — HTTP API (Autodiscover, policy, audit).
//
// Rust mail-core (SMTP/IMAP) — отдельный процесс; см. core/.
// Refs: ADR-0027, ADR-0029, PRD-Comms-MVP.
package main

import (
	"log"
	"net/http"
	"os"

	"era/services/comms/mail/internal/api"
	"era/services/comms/mail/internal/audit"
	"era/services/comms/mail/internal/caladapter"
	"era/services/comms/mail/internal/coreclient"
	"era/services/comms/mail/internal/policy"
	"era/services/comms/mail/internal/repo"
	"era/services/platform/httpserver"
	"era/services/platform/licensegate"
	"era/services/platform/tenant"
)

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)

	gate := licensegate.FromEnv()
	if !gate.Allow(licensegate.ModuleCommsMailServer) {
		if os.Getenv("ERA_MAIL_DEV") == "1" {
			gate = licensegate.FromModules([]licensegate.Module{licensegate.ModuleCommsMailServer})
			log.Print("ERA_MAIL_DEV=1 — comms-mail-server license bypass")
		} else {
			log.Fatal("license: comms-mail-server not enabled (set ERA_LICENSE_MODULES or ERA_MAIL_DEV=1)")
		}
	}

	tenants := tenant.NewStore()
	seedDemoTenants(tenants)

	policies := policy.NewStore()
	aud := audit.NewFromEnv()

	mailRepo, err := repo.New()
	if err != nil {
		log.Fatalf("repo: %v", err)
	}
	seedDemoPolicy(mailRepo)
	seedDemoMailbox(mailRepo)

	calStore := caladapter.New(mailRepo)
	core := coreclient.NewFromEnv()
	useTLS := os.Getenv("ERA_MAIL_TLS") == "1"

	srv := api.NewServer(api.Config{
		Tenants:  tenants,
		Policies: policies,
		Audit:    aud,
		Core:     core,
		Gate:     gate,
		Repo:     mailRepo,
		CalStore: calStore,
		HTTPPort: parsePort(env("ERA_MAIL_HTTP_ADDR", ":8150")),
		UseTLS:   useTLS,
	})

	addr := env("ERA_MAIL_HTTP_ADDR", ":8150")
	mux := http.NewServeMux()
	srv.Register(mux)

	log.Printf("era-mail-api listening %s (storage=%s)", addr, storageMode(mailRepo))
	if err := httpserver.Listen(addr, mux); err != nil {
		log.Fatalf("listen: %v", err)
	}
}

func seedDemoTenants(s *tenant.Store) {
	_ = s.PutTenant(tenant.Tenant{ID: "t-demo", Name: "Demo Gov", Slug: "demo"})
	_ = s.PutDomain(tenant.Domain{ID: "d-mail", TenantID: "t-demo", FQDN: "mail.gov.az", Primary: true})
}

func seedDemoPolicy(r repo.Backend) {
	r.PutPolicy("t-demo", repo.InlinePolicy{
		MaxAttachmentSizeMB:      25,
		QuotaMBPerUser:           512,
		RetentionDays:            365,
		MaxAttachmentsPerMessage: 50,
	})
}

func seedDemoMailbox(r repo.Backend) {
	if _, err := r.GetMailboxByEmail("alice@mail.gov.az"); err == nil {
		return
	}
	_, _ = r.CreateMailbox("t-demo", "alice@mail.gov.az", "demo-pass", 512<<20)
}

func storageMode(r *repo.Repository) string {
	if _, ok := r.Backend.(*repo.Postgres); ok {
		return "postgres"
	}
	return "memory"
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func parsePort(addr string) int {
	if addr == "" {
		return 8150
	}
	if addr[0] == ':' {
		addr = addr[1:]
	}
	var p int
	for _, ch := range addr {
		if ch < '0' || ch > '9' {
			return 8150
		}
		p = p*10 + int(ch-'0')
	}
	if p == 0 {
		return 8150
	}
	return p
}
