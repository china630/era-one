// Package ingest — REST-клиент к ingest-gateway (Path A feed).
package ingest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	erav1 "era/contracts/gen/era/v1"
	"github.com/google/uuid"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	TenantID   string
}

func New(baseURL, tenantID string) *Client {
	return &Client{
		BaseURL: baseURL, TenantID: tenantID,
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	}
}

type batchReq struct {
	BatchID  string            `json:"batch_id"`
	AgentID  string            `json:"agent_id"`
	TenantID string            `json:"tenant_id"`
	Events   []*erav1.Envelope `json:"events"`
}

type batchResp struct {
	Status   string `json:"status"`
	Accepted int    `json:"accepted"`
	Message  string `json:"message,omitempty"`
}

func (c *Client) PostEvents(ctx context.Context, events []*erav1.Envelope) error {
	if c == nil || c.BaseURL == "" || len(events) == 0 {
		return nil
	}
	body, err := json.Marshal(batchReq{
		BatchID: uuid.NewString(), AgentID: "era-observe", TenantID: c.TenantID, Events: events,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/v1/ingest", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var ack batchResp
	_ = json.NewDecoder(resp.Body).Decode(&ack)
	if resp.StatusCode >= 300 || ack.Status == "REJECTED" {
		return fmt.Errorf("ingest %s: %s", ack.Status, ack.Message)
	}
	return nil
}
