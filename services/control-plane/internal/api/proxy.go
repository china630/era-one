package api

import (
	"io"
	"net/http"
	"os"
	"strings"
)

// SOC portal BFF: same-origin proxy to backend services (avoids browser CORS in demo).
func (s *Server) mountProxies(mux *http.ServeMux) {
	mux.HandleFunc("/api/proxy/events", s.proxyGET(serviceURL("ERA_EVENT_WRITER_URL", "http://127.0.0.1:8089"), "/api/events"))
	mux.HandleFunc("/api/proxy/timeline", s.proxyGET(serviceURL("ERA_EVENT_WRITER_URL", "http://127.0.0.1:8089"), "/api/timeline"))
	mux.HandleFunc("/api/proxy/exposure", s.proxyGET(serviceURL("ERA_DETECTION_URL", "http://127.0.0.1:8097"), "/api/v1/exposure"))
	mux.HandleFunc("/api/v1/workbench/timeline", s.handleWorkbenchTimeline)
	mux.HandleFunc("/api/v1/exposure", s.handleExposureProxy)
	mux.HandleFunc("/api/proxy/soar/actions", s.proxyGET(serviceURL("ERA_SOAR_URL", "http://127.0.0.1:8092"), "/api/v1/actions"))
	mux.HandleFunc("/api/proxy/ai/investigate", s.proxyPOST(serviceURL("ERA_AI_CORE_URL", "http://127.0.0.1:8091"), "/api/v1/investigate"))
	mux.HandleFunc("/api/proxy/ingest/healthz", s.proxyGET(serviceURL("ERA_INGEST_URL", "http://127.0.0.1:8082"), "/healthz"))
}
func serviceURL(envKey, def string) string {
	if u := os.Getenv(envKey); u != "" {
		return strings.TrimRight(u, "/")
	}
	return def
}

func (s *Server) proxyGET(base, path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
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
