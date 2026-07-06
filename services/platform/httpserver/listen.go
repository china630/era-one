// Package httpserver — единый HTTP(S) listen с tlsutil (Stage 10d).
package httpserver

import (
	"net/http"

	"era/services/platform/tlsutil"
)

// Listen запускает HTTP или mTLS-сервер в зависимости от ERA_TLS_*.
func Listen(addr string, handler http.Handler) error {
	cfg := tlsutil.ServerFromEnv()
	srv := cfg.HTTPServer(addr, handler)
	return cfg.Listen(srv)
}
