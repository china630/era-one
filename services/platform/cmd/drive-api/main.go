// ERA Drive API — metadata + blob HTTP service (Office P0).
package main

import (
	"log"
	"net/http"
	"os"

	driveapi "era/services/platform/drive/api"
	"era/services/platform/drive/blobstore"
	"era/services/platform/drive"
	"era/services/platform/httpserver"
	"era/services/platform/licensegate"
)

func main() {
	store, closeStore, err := drive.OpenFromEnv()
	if err != nil {
		log.Fatalf("drive store: %v", err)
	}
	defer func() {
		if err := closeStore(); err != nil {
			log.Printf("drive store close: %v", err)
		}
	}()
	if _, ok := store.(*drive.PgStore); ok {
		log.Printf("drive-api: metadata store postgres (ERA_OFFICE_DATABASE_URL)")
	} else {
		log.Printf("drive-api: metadata store memory (no ERA_OFFICE_DATABASE_URL)")
	}

	var blobs drive.BlobStore = drive.NewMemoryBlobStore()

	if minioStore, err := blobstore.OpenFromEnv(); err != nil {
		log.Fatalf("minio: %v", err)
	} else if minioStore != nil {
		blobs = blobstore.Adapter{S: minioStore}
		log.Printf("drive-api: MinIO bucket %s", env("ERA_DRIVE_BUCKET", "era-drive"))
	}

	gate, err := licensegate.GateFromEnv(0)
	if err != nil {
		log.Fatalf("license: %v", err)
	}
	if envTruthy("ERA_OFFICE_DEV") {
		if licensegate.StrictMode() {
			log.Fatalf("drive-api: ERA_OFFICE_DEV forbidden when ERA_LICENSE_STRICT/ERA_PRODUCTION set")
		}
		gate = licensegate.OfficeDevGate()
	}

	srv := driveapi.NewServer(driveapi.Config{
		Store:            store,
		Blobs:            blobs,
		Gate:             gate,
		WorkspaceBaseURL: env("ERA_WORKSPACE_BASE_URL", "https://app.customer.local"),
		JWTSecret:        []byte(env("ERA_IDENTITY_JWT_SECRET", "dev-only-change-in-prod")),
		ServiceToken:     os.Getenv("ERA_DRIVE_SERVICE_TOKEN"),
	})
	mux := http.NewServeMux()
	srv.Register(mux)

	addr := env("ERA_DRIVE_HTTP_ADDR", ":8175")
	log.Printf("drive-api listening %s (platform-drive=%v)", addr, gate.Allow(licensegate.ModulePlatformDrive))
	if err := httpserver.Listen(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func envTruthy(k string) bool {
	v := os.Getenv(k)
	return v == "1" || v == "true" || v == "yes"
}
