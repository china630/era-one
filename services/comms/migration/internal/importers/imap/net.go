package imap

import (
	"fmt"
	"os"
	"strings"

	"era/services/comms/internal/imapclient"
)

// NetworkConfig — source IMAP endpoint.
type NetworkConfig struct {
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Username    string `json:"user"`
	PasswordRef string `json:"password_ref"`
	TLS         bool   `json:"tls"`
}

// ResolvePassword loads password from env ref `env:VAR`.
func (c NetworkConfig) ResolvePassword() (string, error) {
	if c.PasswordRef == "" {
		return "", fmt.Errorf("imap: missing password_ref")
	}
	if strings.HasPrefix(c.PasswordRef, "env:") {
		v := os.Getenv(strings.TrimPrefix(c.PasswordRef, "env:"))
		if v == "" {
			return "", fmt.Errorf("imap: env %s empty", c.PasswordRef)
		}
		return v, nil
	}
	return "", fmt.Errorf("imap: unsupported password_ref %q", c.PasswordRef)
}

// ImportNetwork fetches messages from a live IMAP server.
func ImportNetwork(cfg NetworkConfig, folder string) ([]imapclient.Message, error) {
	cl, err := dialNetwork(cfg)
	if err != nil {
		return nil, err
	}
	defer cl.Close()
	if folder == "" {
		folder = "INBOX"
	}
	return cl.FetchFolder(folder)
}

// ImportNetworkAll fetches all selectable mailboxes (G3).
func ImportNetworkAll(cfg NetworkConfig) ([]imapclient.Message, error) {
	cl, err := dialNetwork(cfg)
	if err != nil {
		return nil, err
	}
	defer cl.Close()
	mboxes, err := cl.ListMailboxesDetailed()
	if err != nil {
		return nil, err
	}
	var out []imapclient.Message
	for _, mb := range mboxes {
		if !mb.Selectable() {
			continue
		}
		msgs, err := cl.FetchFolder(mb.Name)
		if err != nil {
			return nil, fmt.Errorf("imap: fetch %q: %w", mb.Name, err)
		}
		out = append(out, msgs...)
	}
	return out, nil
}

func dialNetwork(cfg NetworkConfig) (*imapclient.Client, error) {
	pass, err := cfg.ResolvePassword()
	if err != nil {
		return nil, err
	}
	port := cfg.Port
	if port == 0 {
		port = 993
	}
	return imapclient.Dial(imapclient.Config{
		Host:     cfg.Host,
		Port:     port,
		Username: cfg.Username,
		Password: pass,
		TLS:      cfg.TLS || port == 993,
	})
}
