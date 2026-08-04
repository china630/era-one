package api

import (
	"net/http"
	"os"
)

func (s *Server) readyz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ready := true
	checks := map[string]string{"repo": "ok"}
	if s.cfg.Repo != nil {
		if p, ok := s.cfg.Repo.(interface{ Ping() error }); ok {
			if err := p.Ping(); err != nil {
				ready = false
				checks["postgres"] = err.Error()
			} else {
				checks["postgres"] = "ok"
			}
		} else {
			checks["postgres"] = "memory"
		}
		if b, ok := s.cfg.Repo.(interface{ PingBlob() error }); ok {
			if err := b.PingBlob(); err != nil {
				ready = false
				checks["minio"] = err.Error()
			} else if checks["minio"] == "" {
				checks["minio"] = "ok"
			}
		} else {
			checks["minio"] = "disabled"
		}
	}
	auditRequired := os.Getenv("ERA_MAIL_AUDIT_REQUIRE") == "1"
	if s.cfg.Audit != nil && s.cfg.Audit.IsConfigured() {
		if err := s.cfg.Audit.Ping(r.Context()); err != nil {
			ready = false
			checks["clickhouse"] = err.Error()
		} else {
			checks["clickhouse"] = "ok"
		}
	} else if auditRequired {
		ready = false
		checks["clickhouse"] = "required_but_disabled"
	} else {
		checks["clickhouse"] = "disabled"
	}
	status := http.StatusOK
	if !ready {
		status = http.StatusServiceUnavailable
	}
	w.WriteHeader(status)
	writeJSON(w, map[string]any{"ready": ready, "checks": checks})
}
