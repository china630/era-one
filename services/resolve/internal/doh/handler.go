// Package doh — DNS-over-HTTPS (RFC 8484 lab endpoint).
package doh

import (
	"encoding/base64"
	"io"
	"net/http"

	"era/services/resolve/internal/dnsx"
)

const maxBody = 65535

// Handler serves /dns-query using the same Guard as UDP DNS.
type Handler struct {
	DNS *dnsx.Server
	// Enabled when false returns 503 (UI DoH toggle).
	Enabled func() bool
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.Enabled != nil && !h.Enabled() {
		http.Error(w, `{"error":"doh disabled"}`, http.StatusServiceUnavailable)
		return
	}
	if h.DNS == nil || h.DNS.Guard == nil {
		http.Error(w, "dns unavailable", http.StatusBadGateway)
		return
	}
	var wire []byte
	var err error
	switch r.Method {
	case http.MethodGet:
		q := r.URL.Query().Get("dns")
		if q == "" {
			http.Error(w, "missing dns=", http.StatusBadRequest)
			return
		}
		wire, err = base64.RawURLEncoding.DecodeString(q)
		if err != nil {
			wire, err = base64.URLEncoding.DecodeString(q)
		}
		if err != nil {
			http.Error(w, "bad dns param", http.StatusBadRequest)
			return
		}
	case http.MethodPost:
		ct := r.Header.Get("Content-Type")
		if ct != "" && ct != "application/dns-message" {
			http.Error(w, "unsupported content-type", http.StatusUnsupportedMediaType)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxBody)
		wire, err = io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "body read", http.StatusBadRequest)
			return
		}
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	resp, err := h.DNS.HandleMessage(wire)
	if err != nil || len(resp) == 0 {
		http.Error(w, "bad query", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/dns-message")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp)
}
