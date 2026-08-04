package main

import (
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"time"

	"era/services/platform/envelope"
	"era/services/platform/licensegate"
	"era/services/resolve/internal/api"
	"era/services/resolve/internal/atlas"
	"era/services/resolve/internal/dnsx"
	"era/services/resolve/internal/guard"
	"era/services/resolve/internal/policy"
	"era/services/resolve/internal/trace"
)

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)
	addr := env("ERA_HTTP_ADDR", ":8134")
	dnsAddr := env("ERA_RESOLVE_DNS_ADDR", ":5353")
	dohAddr := env("ERA_RESOLVE_DOH_ADDR", ":8443")

	pol := policy.NewStore()
	if p := env("ERA_RESOLVE_POLICY_PATH", ""); p != "" {
		if err := pol.LoadFile(p); err != nil {
			log.Printf("policy load: %v", err)
		}
	}
	atl := atlas.New()
	if p := env("ERA_RESOLVE_ATLAS_PATH", ""); p != "" {
		if err := atl.LoadFile(p); err != nil {
			log.Printf("atlas load: %v", err)
		}
	}
	if d := env("ERA_ATLAS_PACK_DIR", ""); d != "" {
		if _, err := atl.LoadFromDir(d); err != nil {
			log.Printf("atlas pack dir: %v", err)
		}
	}
	var pub *envelope.Publisher
	if brokers := env("ERA_KAFKA_BROKERS", ""); brokers != "" {
		pub = envelope.New(envelope.MustBrokers(brokers), env("ERA_TENANT_ID", "tenant-dev"), env("ERA_NODE_ID", "resolve-01"), "resolve")
		defer pub.Close()
	}
	tr := trace.New(256, pub)
	eng := guard.New(pol, atl)
	gate, err := licensegate.GateFromEnv(0)
	if err != nil {
		log.Fatalf("license: %v", err)
	}

	dnsSrv := &dnsx.Server{Guard: eng, Trace: tr, Addr: dnsAddr}
	srv := api.New(eng, pol, atl, tr, gate)
	srv.UIDir = api.ResolveUIDir()
	srv.DNS = dnsSrv

	if env("ERA_RESOLVE_DNS_DISABLE", "") != "1" {
		go func() {
			log.Printf("resolve DNS listening %s", dnsAddr)
			if err := dnsSrv.ListenAndServe(); err != nil {
				log.Printf("dns serve: %v", err)
			}
		}()
	}

	if env("ERA_RESOLVE_DOH_DISABLE", "") != "1" {
		go func() {
			dohHandler := srv.Routes()
			httpsSrv := &http.Server{Addr: dohAddr, Handler: dohHandler, ReadHeaderTimeout: 5 * time.Second}
			cert := env("ERA_RESOLVE_DOH_CERT", "")
			key := env("ERA_RESOLVE_DOH_KEY", "")
			if cert != "" && key != "" {
				log.Printf("resolve DoH HTTPS listening %s", dohAddr)
				if err := httpsSrv.ListenAndServeTLS(cert, key); err != nil {
					log.Printf("doh tls: %v", err)
				}
				return
			}
			if tc := env("ERA_TLS_CERT", ""); tc != "" {
				httpsSrv.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
				log.Printf("resolve DoH TLS listening %s", dohAddr)
				if err := httpsSrv.ListenAndServeTLS(tc, env("ERA_TLS_KEY", "")); err != nil {
					log.Printf("doh: %v", err)
				}
				return
			}
			log.Printf("resolve DoH HTTP (lab) listening %s/dns-query", dohAddr)
			if err := httpsSrv.ListenAndServe(); err != nil {
				log.Printf("doh: %v", err)
			}
		}()
	}

	httpSrv := &http.Server{Addr: addr, Handler: srv.Routes(), ReadHeaderTimeout: 5 * time.Second}
	log.Printf("resolve HTTP listening %s", addr)
	log.Fatal(httpSrv.ListenAndServe())
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
