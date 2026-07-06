package api

import (
	"net/http"
	"strings"
)

func noCacheUI(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".html") || strings.HasSuffix(r.URL.Path, "/") {
			w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
		}
		next.ServeHTTP(w, r)
	})
}
