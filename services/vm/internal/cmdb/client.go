// Package cmdb — чтение asset_software из control-plane для CVE-сверки (ADR-0011).
package cmdb

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type Client struct {
	Base   string
	client *http.Client
}

type SoftwareRow struct {
	NodeID  string `json:"node_id"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

func NewFromEnv() *Client {
	base := os.Getenv("ERA_CONTROL_PLANE_URL")
	if base == "" {
		return nil
	}
	return &Client{Base: strings.TrimRight(base, "/"), client: &http.Client{Timeout: 15 * time.Second}}
}

func (c *Client) ListSoftware() ([]SoftwareRow, error) {
	if c == nil {
		return nil, fmt.Errorf("control-plane not configured")
	}
	req, err := http.NewRequest(http.MethodGet, c.Base+"/api/v1/cmdb/software", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-ERA-Actor", "vm-engine")
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
		Software []SoftwareRow `json:"software"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Software, nil
}

// MatchProducts возвращает установленные пакеты, имя которых содержит product (case-insensitive).
func MatchProducts(rows []SoftwareRow, product string) []SoftwareRow {
	p := strings.ToLower(product)
	var out []SoftwareRow
	for _, r := range rows {
		if strings.Contains(strings.ToLower(r.Name), p) {
			out = append(out, r)
		}
	}
	return out
}
