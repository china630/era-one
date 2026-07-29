// Package internalapi — internal HTTP for Rust mail-core (ADR-0029).
package internalapi

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"era/services/comms/mail/internal/repo"
)

// Handler serves /internal/v1/* for mail-core bridge.
type Handler struct {
	Repo repo.Backend
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/internal/v1/mail/deliver", h.deliver)
	mux.HandleFunc("/internal/v1/mail/list", h.list)
	mux.HandleFunc("/internal/v1/auth/verify", h.verify)
}

type deliverReq struct {
	Email    string `json:"email"`
	From     string `json:"from"`
	Raw      string `json:"raw"`
	RawB64   string `json:"raw_b64"`
}

func (h *Handler) deliver(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req deliverReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	raw := []byte(req.Raw)
	if req.RawB64 != "" {
		b, err := base64.StdEncoding.DecodeString(req.RawB64)
		if err != nil {
			http.Error(w, "bad raw_b64", http.StatusBadRequest)
			return
		}
		raw = b
	}
	msg, err := h.Repo.DeliverRaw(strings.ToLower(req.Email), raw, req.From)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]any{"id": msg.ID, "uid": msg.UID})
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	email := r.URL.Query().Get("email")
	if email == "" {
		http.Error(w, "email required", http.StatusBadRequest)
		return
	}
	msgs, err := h.Repo.ListMessages(email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	type item struct {
		ID   int64  `json:"id"`
		UID  int64  `json:"uid"`
		Raw  string `json:"raw"`
		From string `json:"from"`
	}
	out := make([]item, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, item{ID: m.ID, UID: m.UID, Raw: string(m.Raw), From: m.FromAddr})
	}
	writeJSON(w, map[string]any{"messages": out})
}

type verifyReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) verify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var req verifyReq
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	ok := h.Repo.VerifyMailboxPassword(req.Email, req.Password)
	writeJSON(w, map[string]bool{"ok": ok})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
