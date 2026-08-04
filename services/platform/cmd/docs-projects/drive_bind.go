package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const erajContentType = "application/vnd.era.eraj"

// ErajProject is the Drive-native Projects board format (.eraj).
type ErajProject struct {
	Format        string `json:"format"`
	Name          string `json:"name"`
	DriveObjectID string `json:"drive_object_id,omitempty"`
	TenantID      string `json:"tenant_id,omitempty"`
	Tasks         []task `json:"tasks"`
}

func emptyEraj(name, tenant string) ErajProject {
	name = strings.TrimSpace(name)
	if name == "" {
		name = uniqueErajName("Untitled")
	} else if !strings.HasSuffix(strings.ToLower(name), ".eraj") {
		name = name + ".eraj"
	}
	return ErajProject{
		Format:   "eraj",
		Name:     name,
		TenantID: tenant,
		Tasks:    []task{},
	}
}

func uniqueErajName(base string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		base = "board"
	}
	base = strings.TrimSuffix(base, filepath.Ext(base))
	var b [4]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("%s-%s.eraj", base, hex.EncodeToString(b[:]))
}

type erajDrive interface {
	putEraj(tenantID, userID, name string, doc ErajProject, objectID string) (string, error)
	getEraj(tenantID, userID, objectID string) (ErajProject, error)
}

type driveClient struct {
	base   string
	token  string
	client *http.Client
}

func newDriveClientFromEnv() (erajDrive, error) {
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("ERA_DRIVE_API_URL")), "/")
	token := strings.TrimSpace(os.Getenv("ERA_DRIVE_SERVICE_TOKEN"))
	if base == "" {
		return nil, fmt.Errorf("ERA_DRIVE_API_URL required for .eraj")
	}
	if token == "" {
		return nil, fmt.Errorf("ERA_DRIVE_SERVICE_TOKEN required (fail-closed)")
	}
	return &driveClient{
		base:  base,
		token: token,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}, nil
}

func (c *driveClient) putEraj(tenantID, userID, name string, doc ErajProject, objectID string) (string, error) {
	doc.Format = "eraj"
	raw, err := json.Marshal(doc)
	if err != nil {
		return "", err
	}
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("file", name)
	if err != nil {
		return "", err
	}
	if _, err := part.Write(raw); err != nil {
		return "", err
	}
	_ = w.WriteField("name", name)
	_ = w.WriteField("content_type", erajContentType)
	if err := w.Close(); err != nil {
		return "", err
	}
	url := c.base + "/api/v1/drive/objects"
	if strings.TrimSpace(objectID) != "" {
		url = c.base + "/api/v1/drive/objects/" + objectID + "/versions"
	}
	req, err := http.NewRequest(http.MethodPost, url, &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("X-ERA-Tenant", tenantID)
	req.Header.Set("X-ERA-User", userID)
	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("drive upload %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	if strings.TrimSpace(objectID) != "" {
		return objectID, nil
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return "", err
	}
	id := stringifyJSONField(out, "id", "ID")
	if id == "" {
		return "", fmt.Errorf("drive upload: empty id")
	}
	return id, nil
}

func (c *driveClient) getEraj(tenantID, userID, objectID string) (ErajProject, error) {
	req, err := http.NewRequest(http.MethodGet, c.base+"/api/v1/drive/objects/"+objectID, nil)
	if err != nil {
		return ErajProject{}, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("X-ERA-Tenant", tenantID)
	req.Header.Set("X-ERA-User", userID)
	resp, err := c.client.Do(req)
	if err != nil {
		return ErajProject{}, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return ErajProject{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ErajProject{}, fmt.Errorf("drive get %d", resp.StatusCode)
	}
	var doc ErajProject
	if err := json.Unmarshal(b, &doc); err != nil {
		return ErajProject{}, err
	}
	if doc.Format == "" {
		doc.Format = "eraj"
	}
	if doc.Tasks == nil {
		doc.Tasks = []task{}
	}
	doc.DriveObjectID = objectID
	return doc, nil
}

func stringifyJSONField(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			switch t := v.(type) {
			case string:
				if t != "" {
					return t
				}
			}
		}
	}
	return ""
}

type memErajDrive struct {
	mu   sync.Mutex
	blob map[string][]byte
	meta map[string]string // id -> name
}

func newMemErajDrive() *memErajDrive {
	return &memErajDrive{
		blob: map[string][]byte{},
		meta: map[string]string{},
	}
}

func (m *memErajDrive) putEraj(_tenant, _user, name string, doc ErajProject, objectID string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	doc.Format = "eraj"
	raw, err := json.Marshal(doc)
	if err != nil {
		return "", err
	}
	id := strings.TrimSpace(objectID)
	if id == "" {
		var b [8]byte
		_, _ = rand.Read(b[:])
		id = "eraj-" + hex.EncodeToString(b[:])
	}
	m.blob[id] = raw
	m.meta[id] = name
	return id, nil
}

func (m *memErajDrive) getEraj(_tenant, _user, objectID string) (ErajProject, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	raw, ok := m.blob[objectID]
	if !ok {
		return ErajProject{}, errNotFound
	}
	var doc ErajProject
	if err := json.Unmarshal(raw, &doc); err != nil {
		return ErajProject{}, err
	}
	if doc.Tasks == nil {
		doc.Tasks = []task{}
	}
	doc.DriveObjectID = objectID
	if doc.Format == "" {
		doc.Format = "eraj"
	}
	return doc, nil
}

type erajSessionCache struct {
	mu   sync.Mutex
	docs map[string]*ErajProject
}

func newErajSessionCache() *erajSessionCache {
	return &erajSessionCache{docs: map[string]*ErajProject{}}
}
