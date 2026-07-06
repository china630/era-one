package main

import (
	"log"
	"os"
	"strings"

	"era/services/platform/cpclient"
	"era/services/platform/httpserver"
	"era/services/platform/licensegate"
	"era/services/provision/internal/api"
	minioclient "era/services/provision/internal/minio"
	"era/services/provision/internal/store"
)

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)
	addr := env("ERA_HTTP_ADDR", ":8124")
	st := store.NewMemory()
	if ep := env("ERA_MINIO_ENDPOINT", ""); ep != "" {
		mc, err := minioclient.NewFromEnv()
		if err != nil {
			log.Printf("minio client: %v (static catalog only)", err)
		} else {
			bucket := env("ERA_MINIO_BUCKET", "era-provision")
			prefix := env("ERA_MINIO_PREFIX", "images/")
			st = store.NewMinIOStore(st, mc, bucket, prefix)
			log.Printf("minio image sync enabled bucket=%s", bucket)
		}
	}
	cp := cpclient.New(env("ERA_CONTROL_PLANE_URL", "http://127.0.0.1:8090"))
	srv := api.New(st, licensegate.DevDefault(), cp)
	log.Printf("provision listening %s (MinIO ref: %s)", addr, env("ERA_MINIO_ENDPOINT", "http://127.0.0.1:9000"))
	log.Fatal(httpserver.Listen(addr, srv.Routes()))
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return strings.TrimSpace(v)
	}
	return def
}
