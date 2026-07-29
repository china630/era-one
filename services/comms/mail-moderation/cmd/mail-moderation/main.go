package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"era/services/comms/mail-moderation/internal/adminapi"
	"era/services/comms/mail-moderation/internal/audit"
	"era/services/comms/mail-moderation/internal/engine"
	"era/services/comms/mail-moderation/internal/hold"
	"era/services/comms/mail-moderation/internal/notify"
	"era/services/comms/mail-moderation/internal/policy"
	"era/services/comms/mail-moderation/internal/resolve"
	"era/services/comms/mail-moderation/internal/rules"
	"era/services/comms/mail-moderation/internal/smtpproxy"
	"era/services/platform/httpserver"
	"era/services/platform/licensegate"
)

func main() {
	gate, err := licensegate.GateFromEnv(1)
	if err != nil {
		log.Fatalf("license: %v", err)
	}
	if !gate.Allow(licensegate.ModuleCommsMailModeration) {
		if licensegate.StrictMode() {
			log.Fatal("license: comms-mail-moderation not enabled")
		}
		log.Print("mail-moderation: license module absent; continuing in dev mode")
	}

	memAudit := &audit.Memory{}
	var rec audit.Recorder = memAudit
	if ch := audit.NewCHFromEnv(); ch != nil {
		rec = &audit.Composite{Sinks: []audit.Recorder{memAudit, ch}}
		log.Print("mail-moderation: ClickHouse audit enabled")
	}

	holds, err := hold.OpenRepositoryFromEnv()
	if err != nil {
		log.Fatalf("holds: %v", err)
	}

	baseDir := resolve.DirectoryFromEnv()
	curators, err := resolve.OpenCuratorsFromEnv()
	if err != nil {
		log.Fatalf("curators: %v", err)
	}
	dir := resolve.Directory(baseDir)
	if curators != nil {
		dir = &resolve.OverlayDir{Base: baseDir, Curators: curators}
		log.Print("mail-moderation: PG curators enabled")
	}

	groups := engine.GroupLookup(engine.StaticGroups{})
	if ldap, ok := baseDir.(*resolve.LDAPDir); ok {
		groups = ldap
	}

	tok := notify.NewTokens([]byte(os.Getenv("ERA_MM_TOKEN_SECRET")))
	mailer := notify.MailerFromEnv(env("ERA_MM_NOTIFY_SMTP", os.Getenv("ERA_MM_UPSTREAM")))
	eng := &engine.Engine{
		Local:    splitCSV(env("ERA_MM_LOCAL_DOMAINS", "company.local")),
		Groups:   groups,
		Resolve:  &resolve.Resolver{Dir: dir},
		Holds:    holds,
		Notify:   &notify.Service{From: env("ERA_MM_FROM", "moderation@localhost"), PublicBase: env("ERA_MM_PUBLIC_BASE", "http://127.0.0.1:8360"), Mailer: mailer, Tokens: tok},
		Audit:    rec,
		Upstream: engine.UpstreamFromEnv(os.Getenv("ERA_MM_UPSTREAM")),
	}

	persist, err := rules.OpenFromEnv()
	if err != nil {
		log.Fatalf("rules store: %v", err)
	}
	var rulesPersist adminapi.RulesPersist = &rules.MemorySave{}
	if persist != nil {
		rulesPersist = persist
		if doc, err := persist.LoadDocument(); err == nil && len(doc.Rules) > 0 {
			eng.Rules = doc.Rules
			log.Printf("mail-moderation: loaded %d rules from PG", len(doc.Rules))
		}
	}
	if path := os.Getenv("ERA_MM_RULES_YAML"); path != "" {
		doc, err := policy.LoadDocument(path)
		if err != nil {
			log.Fatalf("rules: %v", err)
		}
		eng.Rules = doc.Rules
		_ = rulesPersist.SaveDocument(doc)
	}

	go func() {
		t := time.NewTicker(time.Minute)
		defer t.Stop()
		for range t.C {
			eng.RunTTL()
		}
	}()

	api := adminapi.New(eng, tok)
	api.Persist = rulesPersist
	api.Curators = curators
	mux := http.NewServeMux()
	api.Register(mux)

	smtpAddr := env("ERA_MM_SMTP_ADDR", ":2525")
	smtp := &smtpproxy.Server{Engine: eng}
	if err := smtp.Listen(smtpAddr); err != nil {
		log.Fatalf("smtp: %v", err)
	}
	log.Printf("comms-mail-moderation smtp %s", smtpAddr)

	httpAddr := env("ERA_MM_HTTP_ADDR", ":8360")
	log.Printf("comms-mail-moderation http %s", httpAddr)
	if err := httpserver.Listen(httpAddr, mux); err != nil {
		log.Fatal(err)
	}
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range splitComma(s) {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func splitComma(s string) []string {
	var cur string
	var out []string
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			out = append(out, trim(cur))
			cur = ""
			continue
		}
		cur += string(s[i])
	}
	out = append(out, trim(cur))
	return out
}

func trim(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}
