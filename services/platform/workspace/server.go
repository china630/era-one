// Package workspace — ERA Workspace BFF shell (Office P0).
package workspace

import (
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"
)

// Config for workspace BFF.
type Config struct {
	DriveAPIURL         string
	IdentityAPIURL      string
	MailUIURL           string
	DocsAPIURL          string
	TablesAPIURL        string
	PresentationsAPIURL string
	ProjectsAPIURL      string
	DocsAIURL           string
	DriveUI             http.Handler
	DocsUI              http.Handler
	TablesUI            http.Handler
	PresentationsUI     http.Handler
	ProjectsUI          http.Handler
	OfficeAIUI          http.Handler
	// OfficeShellUI serves shared Collab v2 assets at /office-assets/.
	OfficeShellUI http.Handler
	// LoginUI serves the dedicated account page at /login (Google-like).
	LoginUI http.Handler
	// JWTSecret validates Bearer on /api/v1/{drive,docs,tables,...} (O-AUTH).
	JWTSecret []byte
}

// Server is the workspace HTTP shell.
type Server struct {
	cfg Config
}

// NewServer creates a workspace BFF.
func NewServer(cfg Config) *Server {
	return &Server{cfg: cfg}
}

// Register mounts /drive, identity oauth proxy, /mail proxy, /docs stub.
func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("/healthz", s.healthz)
	if s.cfg.OfficeShellUI != nil {
		mux.Handle("/office-assets/", http.StripPrefix("/office-assets", s.cfg.OfficeShellUI))
	}
	if s.cfg.LoginUI != nil {
		mux.Handle("/login", s.cfg.LoginUI)
		mux.Handle("/login/", s.cfg.LoginUI)
	}
	if s.cfg.DriveUI != nil {
		mux.Handle("/drive/", http.StripPrefix("/drive", s.cfg.DriveUI))
		mux.HandleFunc("/drive", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/drive" {
				http.NotFound(w, r)
				return
			}
			http.Redirect(w, r, "/drive/", http.StatusFound)
		})
	}
	if s.cfg.DriveAPIURL != "" {
		mux.Handle("/api/v1/drive/", s.apiAuth(s.reverseProxy(s.cfg.DriveAPIURL)))
	}
	if s.cfg.IdentityAPIURL != "" {
		mux.Handle("/oauth2/", s.reverseProxy(s.cfg.IdentityAPIURL))
		mux.Handle("/.well-known/", s.reverseProxy(s.cfg.IdentityAPIURL))
	}
	if s.cfg.MailUIURL != "" {
		mux.Handle("/mail/", s.reverseProxy(s.cfg.MailUIURL))
		mux.HandleFunc("/mail", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/mail/", http.StatusFound)
		})
	}
	if s.cfg.DocsUI != nil {
		mux.Handle("/docs/", http.StripPrefix("/docs", s.cfg.DocsUI))
		mux.HandleFunc("/docs", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/docs" {
				http.NotFound(w, r)
				return
			}
			http.Redirect(w, r, "/docs/", http.StatusFound)
		})
	}
	if s.cfg.DocsAPIURL != "" {
		mux.Handle("/api/v1/docs", s.apiAuth(s.reverseProxy(s.cfg.DocsAPIURL)))
		mux.Handle("/api/v1/docs/", s.apiAuth(s.reverseProxy(s.cfg.DocsAPIURL)))
	}
	if s.cfg.TablesUI != nil {
		mux.Handle("/tables/", http.StripPrefix("/tables", s.cfg.TablesUI))
		mux.HandleFunc("/tables", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/tables" {
				http.NotFound(w, r)
				return
			}
			http.Redirect(w, r, "/tables/", http.StatusFound)
		})
	} else {
		mux.HandleFunc("/tables", s.stubPage("ERA Tables", "P2 roadmap — .erat spreadsheets", "/tables"))
	}
	if s.cfg.TablesAPIURL != "" {
		mux.Handle("/api/v1/tables", s.apiAuth(s.reverseProxy(s.cfg.TablesAPIURL)))
		mux.Handle("/api/v1/tables/", s.apiAuth(s.reverseProxy(s.cfg.TablesAPIURL)))
	}
	if s.cfg.PresentationsUI != nil {
		mux.Handle("/presentations/", http.StripPrefix("/presentations", s.cfg.PresentationsUI))
		mux.HandleFunc("/presentations", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/presentations/", http.StatusFound)
		})
	}
	if s.cfg.PresentationsAPIURL != "" {
		mux.Handle("/api/v1/presentations", s.apiAuth(s.reverseProxy(s.cfg.PresentationsAPIURL)))
		mux.Handle("/api/v1/presentations/", s.apiAuth(s.reverseProxy(s.cfg.PresentationsAPIURL)))
	}
	if s.cfg.ProjectsUI != nil {
		mux.Handle("/projects/", http.StripPrefix("/projects", s.cfg.ProjectsUI))
		mux.HandleFunc("/projects", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/projects/", http.StatusFound)
		})
	}
	if s.cfg.ProjectsAPIURL != "" {
		mux.Handle("/api/v1/projects", s.apiAuth(s.reverseProxy(s.cfg.ProjectsAPIURL)))
		mux.Handle("/api/v1/projects/", s.apiAuth(s.reverseProxy(s.cfg.ProjectsAPIURL)))
	}
	if s.cfg.OfficeAIUI != nil {
		mux.Handle("/office-ai/", http.StripPrefix("/office-ai", s.cfg.OfficeAIUI))
		mux.HandleFunc("/office-ai", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/office-ai/", http.StatusFound)
		})
	}
	if s.cfg.DocsAIURL != "" {
		mux.Handle("/api/v1/docs-ai/", s.apiAuth(s.reverseProxy(s.cfg.DocsAIURL)))
	}
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = io.WriteString(w, `{"status":"ok","service":"workspace"}`)
}

func (s *Server) stubPage(title, body, onlyPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if onlyPath != "" && r.URL.Path != onlyPath {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, "<!DOCTYPE html><html><head><title>"+title+`</title></head><body><h1>`+title+`</h1><p>`+body+`</p><p><a href="/drive/">ERA Drive</a></p></body></html>`)
	}
}

func (s *Server) apiAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(s.cfg.JWTSecret) == 0 {
			next.ServeHTTP(w, r)
			return
		}
		auth := r.Header.Get("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			// WS may pass access_token query
			if tok := r.URL.Query().Get("access_token"); tok != "" {
				r.Header.Set("Authorization", "Bearer "+tok)
				auth = r.Header.Get("Authorization")
			}
		}
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		tokStr := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		if _, err := parseWorkspaceJWT(tokStr, s.cfg.JWTSecret); err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		// Strip client spoof headers; backends use Authorization.
		r.Header.Del("X-ERA-Tenant")
		r.Header.Del("X-ERA-User")
		r.Header.Del("X-ERA-Groups")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) reverseProxy(base string) http.Handler {
	target, err := url.Parse(strings.TrimRight(base, "/"))
	if err != nil {
		panic(err)
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}
	orig := proxy.Director
	proxy.Director = func(r *http.Request) {
		orig(r)
		r.Host = target.Host
		// Preserve WebSocket upgrade for docs sync (/api/v1/docs/:id/sync).
		if strings.EqualFold(r.Header.Get("Upgrade"), "websocket") ||
			strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade") {
			r.Header.Set("Connection", "Upgrade")
			if r.Header.Get("Upgrade") == "" {
				r.Header.Set("Upgrade", "websocket")
			}
		}
	}
	return proxy
}

func Env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
