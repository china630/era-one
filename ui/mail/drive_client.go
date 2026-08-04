package mail

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// HTTPDriveClient calls drive-api /links/attachment.
type HTTPDriveClient struct {
	BaseURL      string
	Client       *http.Client
	ServiceToken string
	UserJWT      string // preferred: user Bearer from mail session
}

// NewHTTPDriveClient creates a client for drive-api.
func NewHTTPDriveClient(baseURL string) *HTTPDriveClient {
	return &HTTPDriveClient{
		BaseURL:      strings.TrimRight(baseURL, "/"),
		Client:       http.DefaultClient,
		ServiceToken: os.Getenv("ERA_DRIVE_SERVICE_TOKEN"),
	}
}

// CreateAttachmentLink returns a workspace deep link for a Drive object.
func (c *HTTPDriveClient) CreateAttachmentLink(tenantID, objectID string) (string, error) {
	if c == nil || c.BaseURL == "" {
		return "", fmt.Errorf("drive: client not configured")
	}
	body, _ := json.Marshal(map[string]string{
		"tenant_id": tenantID,
		"object_id": objectID,
	})
	req, err := http.NewRequest(http.MethodPost, c.BaseURL+"/api/v1/drive/links/attachment", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	switch {
	case c.UserJWT != "":
		req.Header.Set("Authorization", "Bearer "+c.UserJWT)
	case c.ServiceToken != "":
		req.Header.Set("Authorization", "Bearer "+c.ServiceToken)
		req.Header.Set("X-ERA-Tenant", tenantID)
		req.Header.Set("X-ERA-User", "mail-bff")
	default:
		return "", fmt.Errorf("drive: JWT or ERA_DRIVE_SERVICE_TOKEN required")
	}
	resp, err := c.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("drive: status %d", resp.StatusCode)
	}
	var out struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if out.URL == "" {
		return "", fmt.Errorf("drive: empty url")
	}
	return out.URL, nil
}
