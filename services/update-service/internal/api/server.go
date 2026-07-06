package api

import (
	"crypto/ed25519"
	"encoding/json"
	"net/http"
	"sync"

	"era/services/update-service/internal/build"
	"era/services/update-service/internal/bundle"
	"era/services/platform/metrics"
	"era/services/platform/tlsutil"
)

// Server — ERA Update Service v0 (ADR-0018).
type Server struct {
	mu     sync.RWMutex
	token  string
	claims *bundle.Claims
	priv   ed25519.PrivateKey
	pub    ed25519.PublicKey
}

// New создаёт сервер и подписывает текущий corpus-бандл.
func New(priv ed25519.PrivateKey, pub ed25519.PublicKey) (*Server, error) {
	s := &Server{priv: priv, pub: pub}
	if err := s.refreshBundle(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Server) refreshBundle() error {
	kind := bundleKindFromEnv()
	manifest, err := manifestForKind(kind)
	if err != nil || manifest == nil || len(manifest.Files) == 0 {
		manifest = &bundle.Manifest{Files: []bundle.FileEntry{{Path: "stub.yml", SHA256: "00", Size: 1}}}
	}
	claims := bundle.NewClaims(kind, manifest)
	token, err := bundle.SignClaims(claims, s.priv)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.token = token
	s.claims = claims
	s.mu.Unlock()
	return nil
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.Handle("/metrics", metrics.Handler())
	mux.HandleFunc("/api/v1/bundles/latest", s.handleLatest)
	mux.HandleFunc("/api/v1/bundles/offline", s.handleOffline)
	return mux
}

func (s *Server) handleLatest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.mu.RLock()
	token := s.token
	claims := s.claims
	s.mu.RUnlock()
	writeJSON(w, http.StatusOK, map[string]any{
		"token":  token,
		"claims": claims,
	})
}

func (s *Server) handleOffline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	out := r.URL.Query().Get("path")
	if out == "" {
		out = "bundle.token"
	}
	s.mu.RLock()
	token := s.token
	s.mu.RUnlock()
	if err := build.WriteOfflineBundle(out, token); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"path": out})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// ListenAddr запускает HTTP(S) сервер.
func ListenAddr(addr string, handler http.Handler) error {
	tlsCfg := tlsutil.ServerFromEnv()
	srv := tlsCfg.HTTPServer(addr, handler)
	return tlsCfg.Listen(srv)
}
