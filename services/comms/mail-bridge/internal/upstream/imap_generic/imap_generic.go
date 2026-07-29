package imap_generic

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"era/services/comms/internal/imapclient"
)

// Config — Communigate / generic IMAP upstream.
type Config struct {
	IMAPHost    string
	IMAPPort    int
	IMAPUser    string
	IMAPPassRef string
	SMTPHost    string
	SMTPPort    int
	Mailbox     string
}

func ConfigFromEnv() Config {
	port := 143
	if v := os.Getenv("ERA_BRIDGE_UPSTREAM_CG_IMAP_PORT"); v != "" {
		fmt.Sscanf(v, "%d", &port)
	}
	smtpPort := 25
	if v := os.Getenv("ERA_BRIDGE_UPSTREAM_CG_SMTP_PORT"); v != "" {
		fmt.Sscanf(v, "%d", &smtpPort)
	}
	return Config{
		IMAPHost:    os.Getenv("ERA_BRIDGE_UPSTREAM_CG_IMAP_HOST"),
		IMAPPort:    port,
		IMAPUser:    os.Getenv("ERA_BRIDGE_UPSTREAM_CG_IMAP_USER"),
		IMAPPassRef: os.Getenv("ERA_BRIDGE_UPSTREAM_CG_IMAP_PASSWORD_REF"),
		SMTPHost:    os.Getenv("ERA_BRIDGE_UPSTREAM_CG_SMTP_HOST"),
		SMTPPort:    smtpPort,
	}
}

// Backend translates a subset of EWS to IMAP.
type Backend struct {
	cfg Config
}

func New(cfg Config) *Backend { return &Backend{cfg: cfg} }

func (b *Backend) Name() string { return "imap_generic" }

func (b *Backend) ProxyEWS(ctx context.Context, soapAction string, body []byte, headers http.Header) (int, []byte, error) {
	_ = ctx
	raw := string(body)
	action := soapAction
	if action == "" {
		action = detectAction(raw)
	}
	mailbox := b.cfg.Mailbox
	if mailbox == "" {
		mailbox = headers.Get("X-ERA-Mailbox")
	}
	if mailbox == "" {
		mailbox = b.cfg.IMAPUser
	}

	switch {
	case strings.Contains(action, "FindFolder") || strings.Contains(raw, "FindFolder"):
		folders, err := b.listFolders()
		if err != nil {
			return http.StatusBadGateway, soapFault(err), err
		}
		return http.StatusOK, findFolderResponse(folders), nil
	case strings.Contains(action, "SyncFolderItems") || strings.Contains(raw, "SyncFolderItems"):
		msgs, err := b.syncInbox()
		if err != nil {
			return http.StatusBadGateway, soapFault(err), err
		}
		return http.StatusOK, syncFolderItemsResponse(msgs), nil
	case strings.Contains(action, "CreateItem") || strings.Contains(raw, "CreateItem"):
		subject := extractTag(raw, "Subject")
		msgBody := extractTag(raw, "Body")
		if err := b.appendSent(mailbox, subject, msgBody); err != nil {
			return http.StatusBadGateway, soapFault(err), err
		}
		return http.StatusOK, createItemResponse(1), nil
	default:
		return http.StatusNotImplemented, soapFault(fmt.Errorf("unsupported EWS op %s", action)), fmt.Errorf("unsupported")
	}
}

func (b *Backend) dial() (*imapclient.Client, error) {
	pass, err := resolvePassword(b.cfg.IMAPPassRef)
	if err != nil {
		return nil, err
	}
	port := b.cfg.IMAPPort
	if port == 0 {
		port = 143
	}
	return imapclient.Dial(imapclient.Config{
		Host:     b.cfg.IMAPHost,
		Port:     port,
		Username: b.cfg.IMAPUser,
		Password: pass,
		TLS:      port == 993,
	})
}

func (b *Backend) listFolders() ([]string, error) {
	cl, err := b.dial()
	if err != nil {
		return nil, err
	}
	defer cl.Close()
	return cl.ListMailboxes()
}

func (b *Backend) syncInbox() ([]imapclient.Message, error) {
	cl, err := b.dial()
	if err != nil {
		return nil, err
	}
	defer cl.Close()
	return cl.FetchFolder("INBOX")
}

func (b *Backend) appendSent(mailbox, subject, body string) error {
	cl, err := b.dial()
	if err != nil {
		return err
	}
	defer cl.Close()
	sent := "Sent"
	folders, _ := cl.ListMailboxes()
	for _, f := range folders {
		if strings.EqualFold(f, "Sent") || strings.EqualFold(f, "Sent Items") {
			sent = f
			break
		}
	}
	raw := buildRFC822(subject, body, mailbox)
	return cl.Append(sent, raw, false)
}

func resolvePassword(ref string) (string, error) {
	if ref == "" {
		return "", fmt.Errorf("imap_generic: missing password ref")
	}
	v := os.Getenv(ref)
	if v == "" {
		return "", fmt.Errorf("imap_generic: env %s empty", ref)
	}
	return v, nil
}

func detectAction(raw string) string {
	for _, op := range []string{"FindFolder", "SyncFolderItems", "CreateItem", "GetItem"} {
		if strings.Contains(raw, op) {
			return op
		}
	}
	return ""
}

func soapFault(err error) []byte {
	return soapEnvelope(fmt.Sprintf(`<soap:Fault><faultstring>%s</faultstring></soap:Fault>`, xmlEscape(err.Error())))
}
