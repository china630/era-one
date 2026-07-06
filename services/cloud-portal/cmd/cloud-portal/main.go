package main

import (
	"log"
	"os"

	"era/services/cloud-portal/internal/api"
	"era/services/cloud-portal/internal/store"
)

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)
	addr := env("ERA_HTTP_ADDR", ":8120")
	priv, pub, err := api.LoadKeys()
	if err != nil {
		log.Fatalf("vendor keys: %v", err)
	}
	st := store.New()
	st.UpsertInstallation(&store.Installation{
		DeploymentID: env("ERA_DEPLOYMENT_ID", "deploy-dev"),
		TenantID:     env("ERA_TENANT_ID", "tenant-dev"),
		LicenseID:    env("ERA_LICENSE_ID", "lic-dev"),
		Customer:     "Dev Contour",
	})
	srv := api.New(st, priv, pub)
	log.Printf("cloud-portal listening %s", addr)
	log.Fatal(api.ListenAddr(addr, srv.Routes()))
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
