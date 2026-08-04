// Package coreclient — мост Go mail-api ↔ Rust mail-core (ADR-0027).
package coreclient

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// Status — snapshot состояния mail-core.
type Status struct {
	SMTPReady      bool   `json:"smtp_ready"`
	IMAPReady      bool   `json:"imap_ready"`
	MessagesStored uint64 `json:"messages_stored"`
	Version        string `json:"version"`
}

// Client опрашивает Rust mail-core admin HTTP.
type Client struct {
	addr string
	stub bool
	http *http.Client
}

// NewFromEnv создаёт клиент; ERA_MAIL_CORE_ADDR пуст — stub mode.
func NewFromEnv() *Client {
	addr := os.Getenv("ERA_MAIL_CORE_ADDR")
	if addr == "" {
		return NewStub()
	}
	return &Client{
		addr: addr,
		http: &http.Client{Timeout: 2 * time.Second},
	}
}

// NewStub возвращает клиент со статусом «core не подключён».
func NewStub() *Client {
	return &Client{stub: true}
}

// Status возвращает health mail-core.
func (c *Client) Status() Status {
	if c.stub {
		return Status{
			SMTPReady:      false,
			IMAPReady:      false,
			MessagesStored: 0,
			Version:        "era-mail-core/stub",
		}
	}
	url := fmt.Sprintf("http://%s/api/v1/status", c.addr)
	resp, err := c.http.Get(url)
	if err != nil {
		return Status{Version: "era-mail-core/unreachable"}
	}
	defer resp.Body.Close()
	var st Status
	if err := json.NewDecoder(resp.Body).Decode(&st); err != nil {
		return Status{Version: "era-mail-core/bad-json"}
	}
	return st
}
