// ERA Workspace BFF — unified shell for Drive + Mail (Office P0).
package main

import (
	"log"
	"net/http"

	"era/services/platform/httpserver"
	"era/services/platform/workspace"
	"era/ui/docs"
	"era/ui/drive"
	officeai "era/ui/office-ai"
	officeshell "era/ui/office-shell"
	"era/ui/presentations"
	"era/ui/projects"
	"era/ui/tables"
)

func main() {
	cfg := workspace.Config{
		DriveAPIURL:         workspace.Env("ERA_DRIVE_API_URL", "http://127.0.0.1:8175"),
		IdentityAPIURL:      workspace.Env("ERA_IDENTITY_API_URL", "http://127.0.0.1:8160"),
		MailUIURL:           workspace.Env("ERA_MAIL_UI_URL", ""),
		DocsAPIURL:          workspace.Env("ERA_DOCS_API_URL", ""),
		TablesAPIURL:        workspace.Env("ERA_TABLES_API_URL", "http://127.0.0.1:8143"),
		PresentationsAPIURL: workspace.Env("ERA_PRESENTATIONS_API_URL", "http://127.0.0.1:8144"),
		ProjectsAPIURL:      workspace.Env("ERA_PROJECTS_API_URL", "http://127.0.0.1:8145"),
		DocsAIURL:           workspace.Env("ERA_DOCS_AI_URL", "http://127.0.0.1:8146"),
		DriveUI:             drive.Handler(),
		DocsUI:              docs.Handler(),
		TablesUI:            tables.Handler(),
		PresentationsUI:     presentations.Handler(),
		ProjectsUI:          projects.Handler(),
		OfficeAIUI:          officeai.Handler(),
		OfficeShellUI:       officeshell.Handler(),
		LoginUI:             officeshell.LoginPage(),
		JWTSecret:           []byte(workspace.Env("ERA_IDENTITY_JWT_SECRET", "dev-only-change-in-prod")),
	}
	srv := workspace.NewServer(cfg)
	mux := http.NewServeMux()
	srv.Register(mux)

	addr := workspace.Env("ERA_WORKSPACE_HTTP_ADDR", ":8170")
	log.Printf("workspace listening %s", addr)
	if err := httpserver.Listen(addr, mux); err != nil {
		log.Fatal(err)
	}
}
