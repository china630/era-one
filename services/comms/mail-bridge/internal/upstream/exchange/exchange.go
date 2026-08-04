package exchange

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Backend proxies EWS to on-prem Microsoft Exchange.
type Backend struct {
	BaseURL string
	Client  *http.Client
}

func New(baseURL string) *Backend {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	return &Backend{
		BaseURL: baseURL,
		Client:  &http.Client{Timeout: 120 * time.Second},
	}
}

func FromEnv() *Backend {
	return New(os.Getenv("ERA_BRIDGE_UPSTREAM_EXCHANGE_BASE_URL"))
}

func (b *Backend) Name() string { return "exchange" }

func (b *Backend) ProxyEWS(ctx context.Context, soapAction string, body []byte, headers http.Header) (int, []byte, error) {
	if b.BaseURL == "" {
		return http.StatusBadGateway, nil, fmt.Errorf("exchange: missing base_url")
	}
	url := b.BaseURL + "/ews/Exchange.asmx"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	if soapAction != "" {
		req.Header.Set("SOAPAction", normalizeSOAPAction(soapAction))
	}
	for _, k := range []string{"Authorization", "X-AnchorMailbox", "X-Requested-With"} {
		if v := headers.Get(k); v != "" {
			req.Header.Set(k, v)
		}
	}
	if mb := headers.Get("X-ERA-Mailbox"); mb != "" {
		req.Header.Set("X-AnchorMailbox", mb)
	}
	resp, err := b.Client.Do(req)
	if err != nil {
		return http.StatusBadGateway, nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}
	return resp.StatusCode, respBody, nil
}

func normalizeSOAPAction(action string) string {
	action = strings.Trim(action, `"`)
	if strings.Contains(action, "http") {
		return action
	}
	op := action
	for _, o := range []string{"FindFolder", "SyncFolderItems", "CreateItem", "GetItem", "UpdateItem", "DeleteItem", "GetFolder"} {
		if strings.Contains(action, o) {
			op = o
			break
		}
	}
	return fmt.Sprintf(`"http://schemas.microsoft.com/exchange/services/2006/messages/%s"`, op)
}
