package adapters

import (
	"encoding/json"
	"errors"
)

type ZabbixWebhook struct {
	Host      string `json:"host"`
	HostName  string `json:"hostname"`
	Trigger   string `json:"trigger"`
	Severity  string `json:"severity"`
	EventName string `json:"event_name"`
}

func ParseZabbix(body []byte) (ZabbixWebhook, error) {
	var w ZabbixWebhook
	if err := json.Unmarshal(body, &w); err != nil {
		return w, err
	}
	if w.Host == "" && w.HostName == "" {
		return w, errors.New("host required")
	}
	return w, nil
}

func (w ZabbixWebhook) NodeID() string {
	if w.Host != "" {
		return "net-" + sanitizeID(w.Host)
	}
	return "net-" + sanitizeID(w.HostName)
}

func (w ZabbixWebhook) Summary() string {
	if w.EventName != "" {
		return w.EventName
	}
	return w.Trigger
}
