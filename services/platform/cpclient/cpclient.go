// Package cpclient — HTTP client for control-plane case API (AI, detection, SOAR).
package cpclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	Base   string
	Actor  string
	client *http.Client
}

func New(base string) *Client {
	if base == "" {
		return nil
	}
	return &Client{
		Base:   base,
		Actor:  "era-service",
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) WithActor(actor string) *Client {
	if c == nil {
		return nil
	}
	clone := *c
	if actor != "" {
		clone.Actor = actor
	}
	return &clone
}

func (c *Client) CreateCase(title, detectionID, nodeID string) (string, error) {
	if c == nil {
		return "", fmt.Errorf("control-plane not configured")
	}
	body, _ := json.Marshal(map[string]string{
		"title": title, "detection_id": detectionID, "node_id": nodeID,
	})
	req, err := http.NewRequest(http.MethodPost, c.Base+"/api/v1/cases", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-ERA-Actor", c.Actor)
	req.Header.Set("X-ERA-Role", "admin")
	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("control-plane status %d", resp.StatusCode)
	}
	var out struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return out.ID, nil
}

// Asset — зарегистрированный хост (control-plane).
type Asset struct {
	NodeID   string `json:"node_id"`
	Hostname string `json:"hostname"`
	Platform string `json:"platform"`
}

// ListAssets возвращает активы из control-plane (для exposure criticality).
func (c *Client) ListAssets() ([]Asset, error) {
	if c == nil {
		return nil, fmt.Errorf("control-plane not configured")
	}
	req, err := http.NewRequest(http.MethodGet, c.Base+"/api/v1/assets", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-ERA-Actor", c.Actor)
	req.Header.Set("X-ERA-Role", "admin")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("control-plane status %d", resp.StatusCode)
	}
	var out struct {
		Assets []Asset `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Assets, nil
}

// RegisterAsset — post-install enroll (provision / deploy).
func (c *Client) RegisterAsset(agentID, tenantID, nodeID, hostname, platform, agentVersion string) error {
	if c == nil {
		return fmt.Errorf("control-plane not configured")
	}
	body, _ := json.Marshal(map[string]string{
		"agent_id": agentID, "tenant_id": tenantID, "node_id": nodeID,
		"hostname": hostname, "platform": platform, "agent_version": agentVersion,
	})
	req, err := http.NewRequest(http.MethodPost, c.Base+"/api/v1/assets/register", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-ERA-Actor", c.Actor)
	req.Header.Set("X-ERA-Role", "admin")
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("control-plane status %d", resp.StatusCode)
	}
	return nil
}

// GetAsset returns a single CMDB asset by node_id.
func (c *Client) GetAsset(nodeID string) (*Asset, error) {
	if c == nil {
		return nil, fmt.Errorf("control-plane not configured")
	}
	req, err := http.NewRequest(http.MethodGet, c.Base+"/api/v1/cmdb/assets/"+nodeID, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-ERA-Actor", c.Actor)
	req.Header.Set("X-ERA-Role", "admin")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("asset not found")
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("control-plane status %d", resp.StatusCode)
	}
	var out struct {
		Asset Asset `json:"asset"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out.Asset, nil
}
