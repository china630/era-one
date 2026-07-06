// Package adapters — PRTG/Zabbix/syslog → network events (Vision §8.4).
package adapters

import (
	"encoding/json"
	"errors"
	"strings"
)

// PRTGWebhook — типичный JSON PRTG notification.
type PRTGWebhook struct {
	Device   string `json:"device"`
	Host     string `json:"host"`
	Status   string `json:"status"`
	Message  string `json:"message"`
	Sensor   string `json:"sensor"`
	Priority int    `json:"priority"`
}

func ParsePRTG(body []byte) (PRTGWebhook, error) {
	var w PRTGWebhook
	if err := json.Unmarshal(body, &w); err != nil {
		return w, err
	}
	if w.Device == "" && w.Host == "" {
		return w, errors.New("device or host required")
	}
	return w, nil
}

func (w PRTGWebhook) NodeID() string {
	if w.Host != "" {
		return "net-" + sanitizeID(w.Host)
	}
	return "net-" + sanitizeID(w.Device)
}

func (w PRTGWebhook) Summary() string {
	if w.Message != "" {
		return w.Message
	}
	return w.Sensor + " " + w.Status
}

func (w PRTGWebhook) Detail() string {
	return strings.TrimSpace(w.Device + " " + w.Sensor)
}

func sanitizeID(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, ":", "-")
	s = strings.ReplaceAll(s, ".", "-")
	if len(s) > 48 {
		return s[:48]
	}
	return s
}
