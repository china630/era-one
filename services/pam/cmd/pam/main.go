package main

import (
	"log"
	"os"

	"era/services/platform/privilegedsession"
	"era/services/pam/internal/api"
	"era/services/pam/internal/checkout"
	"era/services/pam/internal/kms"
	"era/services/pam/internal/vault"
	"era/services/platform/httpserver"
	"era/services/platform/licensegate"
)

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)
	addr := env("ERA_HTTP_ADDR", ":8130")
	stateDir := env("ERA_PAM_STATE", os.TempDir()+"/era-pam-state")
	kmsName := env("ERA_KMS_PROVIDER", "software-sealed-dev")
	provider, err := kms.SelectProvider(kmsName, stateDir)
	if err != nil {
		log.Fatalf("kms: %v", err)
	}
	v := vault.New(provider)
	if ps, err := vault.NewPersistStore(stateDir); err != nil {
		log.Printf("pam persist disabled: %v", err)
	} else if err := v.BindPersist(ps); err != nil {
		log.Printf("pam persist load: %v", err)
	}
	ch := checkout.NewStore()
	sess := privilegedsession.NewStore()
	gate := licensegate.DevAllEnabled()
	srv := api.New(v, ch, sess, gate, provider.Name())
	log.Printf("era-pam listening %s state=%s kms=%s", addr, stateDir, provider.Name())
	log.Fatal(httpserver.Listen(addr, srv.Routes()))
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
