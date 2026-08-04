package main

import (
	"log"
	"net/http"
	"os"

	"era/services/platform/httpserver"
	"era/ui/mail"
)

func main() {
	var driveClient mail.DriveClient
	if url := os.Getenv("ERA_DRIVE_API_URL"); url != "" {
		driveClient = mail.NewHTTPDriveClient(url)
		log.Printf("ui-mail: Drive hook enabled -> %s", url)
	}
	mux := http.NewServeMux()
	mail.NewServer(driveClient).Register(mux)
	addr := env("ERA_UI_MAIL_ADDR", ":8180")
	log.Printf("ui-mail listening %s", addr)
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
