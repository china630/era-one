// Package activesync — Exchange ActiveSync WBXML subset (R-GOV-5).
package activesync

import (
	"io"
	"net/http"
	"strings"

	"era/services/comms/mail/internal/repo"
)

// Handler serves /Microsoft-Server-ActiveSync.
type Handler struct {
	Repo repo.Backend
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cmd := strings.ToLower(r.URL.Query().Get("Cmd"))
	user := r.URL.Query().Get("User")
	if user == "" {
		user = "alice@mail.gov.az"
	}
	deviceID := normalizeDeviceID(r.Header.Get("X-MS-DeviceId"))
	if deviceID == "era-device-1" {
		if q := r.URL.Query().Get("DeviceId"); q != "" {
			deviceID = normalizeDeviceID(q)
		}
	}
	body, _ := io.ReadAll(r.Body)
	w.Header().Set("Content-Type", "application/vnd.ms-sync.wbxml")

	switch cmd {
	case "provision":
		w.Write(encodeProvisionResponse("1"))
	case "foldersync":
		h.folderSync(w, body, deviceID, user)
	case "sync":
		h.sync(w, body, deviceID, user)
	case "ping":
		w.Write(encodePingResponse())
	default:
		http.Error(w, "unsupported command", http.StatusNotImplemented)
	}
}

func (h *Handler) mailboxID(email string) string {
	mb, err := h.Repo.GetMailboxByEmail(email)
	if err != nil {
		return email
	}
	return mb.ID
}

func (h *Handler) folderSync(w http.ResponseWriter, body []byte, deviceID, user string) {
	payload, err := parseBody(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	prev := syncKeyFromRequest(payload)
	folderID := "0"
	mailboxID := h.mailboxID(user)
	newKey := nextSyncKey(prev)
	if prev == "" || prev == "0" {
		_ = h.Repo.PutEASSyncKey(deviceID, mailboxID, folderID, newKey)
		w.Write(encodeFolderSyncResponse(newKey))
		return
	}
	if stored, ok := h.Repo.GetEASSyncKey(deviceID, mailboxID, folderID); ok {
		w.Write(encodeFolderSyncResponse(stored))
		return
	}
	_ = h.Repo.PutEASSyncKey(deviceID, mailboxID, folderID, newKey)
	w.Write(encodeFolderSyncResponse(newKey))
}

func (h *Handler) sync(w http.ResponseWriter, body []byte, deviceID, user string) {
	payload, err := parseBody(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	prev := syncKeyFromRequest(payload)
	folderID := folderIDFromRequest(payload)
	mailboxID := h.mailboxID(user)
	newKey := nextSyncKey(prev)
	_ = h.Repo.PutEASSyncKey(deviceID, mailboxID, folderID, newKey)

	msgs, _ := h.Repo.ListMessages(user)
	changeCount := 0
	if prev == "" || prev == "0" {
		changeCount = len(msgs)
	}
	w.Write(encodeSyncResponse(newKey, changeCount))
}
