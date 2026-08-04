package meet

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
)

type Server struct {
	VCSURL string
}

func NewServer() *Server {
	u := os.Getenv("ERA_VCS_URL")
	if u == "" {
		u = "http://127.0.0.1:8270"
	}
	return &Server{VCSURL: strings.TrimRight(u, "/")}
}

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("/meet/healthz", s.healthz)
	mux.HandleFunc("/meet/api/room", s.withRBAC(s.createRoom))
	mux.HandleFunc("/meet/join", s.withRBAC(s.joinPage))
	mux.Handle("/meet/static/", http.StripPrefix("/meet/static/", http.FileServer(http.FS(staticFS()))))
	mux.HandleFunc("/meet", s.withRBAC(s.index))
}

func (s *Server) withRBAC(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-ERA-Tenant") == "" || !strings.Contains(r.Header.Get("X-ERA-Role"), "vcs.user") {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	// Honesty: default client is livekit-stub.js (not real media).
	writeJSON(w, map[string]string{"status": "ok", "service": "ui-meet", "mode": "stub"})
}

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{
		"route":   "/meet",
		"tenant":  r.Header.Get("X-ERA-Tenant"),
		"feature": "meet-room",
		"api":     "/meet/api/room",
		"join":    "/meet/join",
	})
}

func (s *Server) joinPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(joinHTML))
}

func (s *Server) createRoom(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	name := r.URL.Query().Get("name")
	if name == "" {
		name = "era-meet"
	}
	participant := r.URL.Query().Get("identity")
	if participant == "" {
		participant = r.Header.Get("X-ERA-User")
	}
	if participant == "" {
		participant = "user"
	}
	req, err := http.NewRequest(http.MethodPost, s.VCSURL+"/api/v1/vcs/rooms", strings.NewReader(`{"name":"`+name+`"}`))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-ERA-Tenant", r.Header.Get("X-ERA-Tenant"))
	resp, err := http.DefaultClient.Do(req)
	roomID := "lab-" + name
	token := "lk-token-" + name + "-" + participant
	livekitURL := os.Getenv("ERA_LIVEKIT_URL")
	mode := "stub"
	if livekitURL == "" {
		livekitURL = "ws://127.0.0.1:7880"
	} else {
		mode = "live"
	}
	if err == nil && resp != nil {
		defer resp.Body.Close()
		var out map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&out)
		if id, ok := out["id"].(string); ok && id != "" {
			roomID = id
		}
		if t, ok := out["token"].(string); ok && t != "" {
			token = t
		}
		if u, ok := out["url"].(string); ok && u != "" {
			livekitURL = u
			mode = "live"
		}
	}
	if strings.HasPrefix(token, "lk-token-") && mode != "live" {
		mode = "stub"
	}
	writeJSON(w, map[string]any{
		"room_id":     roomID,
		"name":        name,
		"token":       token,
		"participant": participant,
		"vcs_url":     s.VCSURL,
		"livekit_url": livekitURL,
		"mode":        mode,
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
