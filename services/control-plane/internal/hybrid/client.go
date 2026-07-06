package hybrid

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"era/services/control-plane/internal/store"
	"era/services/platform/tlsutil"
)

// EgressRecorder фиксирует исходящие соединения Relay.
type EgressRecorder interface {
	RecordEgressAudit(e *store.EgressAuditEntry)
}

// HTTPDoer выполняет HTTP с allowlist и audit.
type HTTPDoer struct {
	Client    *http.Client
	Allowlist []string
	Recorder  EgressRecorder
	Level     string
}

func NewHTTPDoer(allowlist []string, rec EgressRecorder, level string) *HTTPDoer {
	client := tlsutil.DevHTTPClient(30 * time.Second)
	if c, err := tlsutil.HTTPClient(30 * time.Second); err == nil && c != nil && os.Getenv("ERA_TLS_CA") != "" {
		client = c
	}
	return &HTTPDoer{
		Client:    client,
		Allowlist: allowlist,
		Recorder:  rec,
		Level:     level,
	}
}

func (d *HTTPDoer) Get(rawURL, kind string) ([]byte, int, error) {
	return d.do(http.MethodGet, rawURL, kind, nil)
}

func (d *HTTPDoer) PostJSON(rawURL, kind string, body any) ([]byte, int, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, 0, err
	}
	return d.do(http.MethodPost, rawURL, kind, b)
}

func (d *HTTPDoer) do(method, rawURL, kind string, body []byte) ([]byte, int, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, 0, err
	}
	host := u.Hostname()
	if !HostAllowed(d.Allowlist, host) {
		return nil, 0, fmt.Errorf("hybrid: egress blocked for host %q", host)
	}
	var rdr io.Reader
	if len(body) > 0 {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, rawURL, rdr)
	if err != nil {
		return nil, 0, err
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := d.Client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	hash := sha256.Sum256(data)
	if d.Recorder != nil {
		d.Recorder.RecordEgressAudit(&store.EgressAuditEntry{
			Kind:        kind,
			Target:      host,
			Level:       d.Level,
			Bytes:       len(data),
			PayloadHash: hex.EncodeToString(hash[:8]),
		})
	}
	if resp.StatusCode >= 400 {
		return data, resp.StatusCode, fmt.Errorf("hybrid: http %d from %s", resp.StatusCode, host)
	}
	return data, resp.StatusCode, nil
}

// LeaseResponse — ответ Portal на renew lease.
type LeaseResponse struct {
	Token string `json:"token"`
}

// CRLResponse — ответ Portal на CRL pull.
type CRLResponse struct {
	Token string `json:"token"`
}

// BundleResponse — ответ Update Service.
type BundleResponse struct {
	Token string `json:"token"`
}

func joinURL(base, path string) string {
	return strings.TrimRight(base, "/") + path
}
