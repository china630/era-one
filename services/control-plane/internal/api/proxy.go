package api

import (
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"era/services/control-plane/internal/rbac"
	"era/services/platform/licensegate"
)

// editionProxy maps /api/x/{name}/... → ERA_*_URL (same-origin BFF for Control shell).
type editionProxy struct {
	Name    string
	EnvKey  string
	Default string
}

var editionProxies = []editionProxy{
	{"ai", "ERA_AI_CORE_URL", "http://127.0.0.1:8091"},
	{"soar", "ERA_SOAR_URL", "http://127.0.0.1:8092"},
	{"vm", "ERA_VM_URL", "http://127.0.0.1:8081"},
	{"pam", "ERA_PAM_URL", "http://127.0.0.1:8130"},
	{"observe", "ERA_OBSERVE_URL", "http://127.0.0.1:8132"},
	{"waf", "ERA_WAF_URL", "http://127.0.0.1:8093"},
	{"ngfw", "ERA_NGFW_URL", "http://127.0.0.1:8094"},
	{"resolve", "ERA_RESOLVE_URL", "http://127.0.0.1:8134"},
	{"service", "ERA_SERVICE_DESK_URL", "http://127.0.0.1:8122"},
	{"provision", "ERA_PROVISION_URL", "http://127.0.0.1:8124"},
	{"dlp", "ERA_DLP_URL", "http://127.0.0.1:8095"},
	{"events", "ERA_EVENT_WRITER_URL", "http://127.0.0.1:8089"},
	{"detection", "ERA_DETECTION_URL", "http://127.0.0.1:8097"},
	{"ingest", "ERA_INGEST_URL", "http://127.0.0.1:8082"},
}

// SOC portal BFF: same-origin proxy to backend services (avoids browser CORS in demo).
func (s *Server) mountProxies(mux *http.ServeMux) {
	mux.HandleFunc("/api/proxy/events", s.proxyGET(serviceURL("ERA_EVENT_WRITER_URL", "http://127.0.0.1:8089"), "/api/events"))
	mux.HandleFunc("/api/proxy/timeline", s.proxyGET(serviceURL("ERA_EVENT_WRITER_URL", "http://127.0.0.1:8089"), "/api/timeline"))
	mux.HandleFunc("/api/proxy/exposure", s.proxyGET(serviceURL("ERA_DETECTION_URL", "http://127.0.0.1:8097"), "/api/v1/exposure"))
	mux.HandleFunc("/api/v1/workbench/timeline", s.handleWorkbenchTimeline)
	mux.HandleFunc("/api/v1/workbench/case-bundle", s.handleCaseBundle)
	mux.HandleFunc("/api/v1/exposure", s.handleExposureProxy)
	mux.HandleFunc("/api/v1/exposure/", s.handleExposureByAsset)
	mux.HandleFunc("/api/proxy/soar/actions", s.proxyGET(serviceURL("ERA_SOAR_URL", "http://127.0.0.1:8092"), "/api/v1/actions"))
	mux.HandleFunc("/api/proxy/ai/investigate", s.proxyPOST(serviceURL("ERA_AI_CORE_URL", "http://127.0.0.1:8091"), "/api/v1/investigate"))
	mux.HandleFunc("/api/proxy/ingest/healthz", s.proxyGET(serviceURL("ERA_INGEST_URL", "http://127.0.0.1:8082"), "/healthz"))
	mux.HandleFunc("/api/x/", s.handleEditionProxy)
	mux.HandleFunc("/api/v1/shell/config", s.handleShellConfig)
}

func serviceURL(envKey, def string) string {
	if u := os.Getenv(envKey); u != "" {
		return strings.TrimRight(u, "/")
	}
	return def
}

func (s *Server) handleShellConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	role := rbac.FromRequest(r)
	mods := map[string]bool{
		"core": true,
	}
	pairs := []struct {
		key string
		mod licensegate.Module
	}{
		{"ai", licensegate.ModuleControlAI},
		{"response", licensegate.ModuleResponse},
		{"vm", licensegate.ModuleVuln},
		{"manage", licensegate.ModuleManage},
		{"service", licensegate.ModuleService},
		{"provision", licensegate.ModuleProvision},
		{"pam", licensegate.ModulePAM},
		{"observe", licensegate.ModuleObserve},
		{"perimeter", licensegate.ModulePerimeter},
		{"resolve", licensegate.ModuleResolve},
	}
	for _, p := range pairs {
		if s.Gate == nil {
			mods[p.key] = true
		} else {
			mods[p.key] = s.Gate.Allow(p.mod)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"role":    string(role),
		"trust":   string(rbac.TrustFromEnv()),
		"modules": mods,
		"proxy":   "/api/x/",
		"note":    "Control shell BFF — edition UIs use same-origin /api/x/{svc}/",
	})
}

func (s *Server) handleEditionProxy(w http.ResponseWriter, r *http.Request) {
	role := rbac.FromRequest(r)
	rest := strings.TrimPrefix(r.URL.Path, "/api/x/")
	parts := strings.SplitN(rest, "/", 2)
	if parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	name := parts[0]
	var ep *editionProxy
	for i := range editionProxies {
		if editionProxies[i].Name == name {
			ep = &editionProxies[i]
			break
		}
	}
	if ep == nil {
		http.NotFound(w, r)
		return
	}
	mutating := r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions
	if mutating {
		if !rbac.CanWriteCases(role) && !rbac.IsAdmin(role) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
	} else if !rbac.CanReadCases(role) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	base := serviceURL(ep.EnvKey, ep.Default)
	targetPath := "/"
	if len(parts) > 1 && parts[1] != "" {
		targetPath = "/" + parts[1]
	}
	u, err := url.Parse(base)
	if err != nil {
		http.Error(w, "bad upstream", http.StatusInternalServerError)
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(u)
	orig := proxy.Director
	proxy.Director = func(req *http.Request) {
		orig(req)
		req.URL.Path = targetPath
		req.URL.RawPath = targetPath
		req.Host = u.Host
		// Forward trust for lab backends that may grow AuthZ later.
		if r.Header.Get("X-ERA-Trusted-Proxy") != "" {
			req.Header.Set("X-ERA-Trusted-Proxy", r.Header.Get("X-ERA-Trusted-Proxy"))
		}
		if a := r.Header.Get("X-ERA-Actor"); a != "" {
			req.Header.Set("X-ERA-Actor", a)
		}
		if role != "" {
			req.Header.Set("X-ERA-Role", string(role))
		}
	}
	proxy.ServeHTTP(w, r)
}

func (s *Server) proxyGET(base, path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !rbac.CanReadCases(rbac.FromRequest(r)) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		target := base + path
		if q := r.URL.RawQuery; q != "" {
			target += "?" + q
		}
		resp, err := http.Get(target)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		copyHeader(w.Header(), resp.Header)
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
	}
}

func (s *Server) proxyPOST(base, path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !rbac.CanWriteCases(rbac.FromRequest(r)) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, base+path, r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		req.Header.Set("Content-Type", r.Header.Get("Content-Type"))
		if req.Header.Get("Content-Type") == "" {
			req.Header.Set("Content-Type", "application/json")
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		copyHeader(w.Header(), resp.Header)
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
	}
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		if k == "Connection" {
			continue
		}
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
	if dst.Get("Content-Type") == "" {
		dst.Set("Content-Type", "application/json")
	}
}
