package main

import (
	"log"
	"os"

	"era/services/platform/adminportal"
	"era/services/platform/httpserver"
)

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)
	addr := env("ERA_HTTP_ADDR", ":8140")
	shell := adminportal.NewShell()
	log.Printf("admin-portal listening %s (products=%d)", addr, len(shell.List()))
	log.Fatal(httpserver.Listen(addr, shell.Routes()))
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
