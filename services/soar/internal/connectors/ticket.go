// Package connectors — реальные интеграции SOAR (GA-1, F-GA-12).
package connectors

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// TicketResult — ответ внешней системы тикетов.
type TicketResult struct {
	ExternalID string
	Detail     string
}

// TicketWebhook POST JSON на ERA_SOAR_TICKET_WEBHOOK.
type TicketWebhook struct {
	URL    string
	Client *http.Client
}

func NewTicketWebhook(url string) *TicketWebhook {
	return &TicketWebhook{
		URL: url,
		Client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (w *TicketWebhook) Create(title, caseID string) (TicketResult, error) {
	body, _ := json.Marshal(map[string]string{
		"title":   title,
		"case_id": caseID,
		"source":  "era-xdr-soar",
	})
	req, err := http.NewRequest(http.MethodPost, w.URL, bytes.NewReader(body))
	if err != nil {
		return TicketResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := w.Client.Do(req)
	if err != nil {
		return TicketResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return TicketResult{}, fmt.Errorf("webhook status %d", resp.StatusCode)
	}
	var out struct {
		ID     string `json:"id"`
		Ticket string `json:"ticket_id"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	ext := out.ID
	if ext == "" {
		ext = out.Ticket
	}
	return TicketResult{
		ExternalID: ext,
		Detail:     fmt.Sprintf("ticket created via webhook %s", w.URL),
	}, nil
}

// HostIsolator — изоляция хоста через локальный скрипт (SSH/bastion настраивается в скрипте).
type HostIsolator struct {
	ScriptPath string
}

func (h *HostIsolator) Isolate(nodeID string) (string, error) {
	var cmd *exec.Cmd
	if strings.HasSuffix(strings.ToLower(h.ScriptPath), ".sh") {
		cmd = exec.Command("sh", h.ScriptPath, nodeID)
	} else {
		cmd = exec.Command(h.ScriptPath, nodeID)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("isolate script: %w: %s", err, bytes.TrimSpace(out))
	}
	return string(bytes.TrimSpace(out)), nil
}
