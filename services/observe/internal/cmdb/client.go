// Package cmdb — регистрация сетевых устройств в control-plane CMDB.
package cmdb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type NetworkAsset struct {
	NodeID    string   `json:"node_id"`
	TenantID  string   `json:"tenant_id"`
	Hostname  string   `json:"hostname,omitempty"`
	Kind      string   `json:"kind,omitempty"`
	AssetKind string   `json:"asset_kind,omitempty"`
	Managed   bool     `json:"managed"`
	IPAddrs   []string `json:"ip_addrs,omitempty"`
	MACAddrs  []string `json:"mac_addrs,omitempty"`
}

type reconcileResp struct {
	Asset    NetworkAsset `json:"asset"`
	Conflict bool         `json:"conflict"`
	Audit    string       `json:"audit,omitempty"`
}

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) ReconcileNetwork(ctx context.Context, a NetworkAsset) (reconcileResp, error) {
	var out reconcileResp
	if c == nil || c.BaseURL == "" {
		return out, nil
	}
	body, _ := json.Marshal(a)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/v1/cmdb/network/assets", bytes.NewReader(body))
	if err != nil {
		return out, err
	}
	req.Header.Set("Content-Type", "application/json")
	if a.TenantID != "" {
		req.Header.Set("X-ERA-Tenant-ID", a.TenantID)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return out, fmt.Errorf("cmdb status %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return out, err
	}
	return out, nil
}

func (c *Client) ListNetwork(ctx context.Context) ([]NetworkAsset, error) {
	if c == nil || c.BaseURL == "" {
		return nil, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/api/v1/cmdb/network/assets", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var wrap struct {
		Assets []NetworkAsset `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrap); err != nil {
		return nil, err
	}
	return wrap.Assets, nil
}
