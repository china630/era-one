package autodiscover

import (
	"fmt"
	"strings"
	"text/template"

	"era/services/platform/tenant"
)

// Config for bridge autodiscover responses.
type Config struct {
	Email      string
	BridgeHost string
	HTTPPort   int
	UseTLS     bool
	Tenants    *tenant.Store
}

const autodiscoverTmpl = `<?xml version="1.0" encoding="utf-8"?>
<Autodiscover xmlns="http://schemas.microsoft.com/exchange/autodiscover/responseschema/2006">
  <Response xmlns="http://schemas.microsoft.com/exchange/autodiscover/outlook/responseschema/2006a">
    <Account>
      <AccountType>email</AccountType>
      <Action>settings</Action>
      <Protocol>
        <Type>IMAP</Type>
        <Server>{{ .IMAPHost }}</Server>
        <Port>{{ .IMAPPort }}</Port>
        <LoginName>{{ .Email }}</LoginName>
        <SSL>{{ .IMAPSSL }}</SSL>
      </Protocol>
      <Protocol>
        <Type>EXCH</Type>
        <Server>{{ .EWSHost }}</Server>
        <EwsUrl>https://{{ .EWSHost }}:{{ .HTTPPort }}/ews/Exchange.asmx</EwsUrl>
        <LoginName>{{ .Email }}</LoginName>
        <SSL>{{ .EXCHSSL }}</SSL>
      </Protocol>
    </Account>
  </Response>
</Autodiscover>
`

var tmpl = template.Must(template.New("bridge-autodiscover").Parse(autodiscoverTmpl))

// Render builds Autodiscover XML pointing clients at the bridge façade.
func Render(cfg Config) (string, error) {
	email := strings.ToLower(strings.TrimSpace(cfg.Email))
	if email == "" || !strings.Contains(email, "@") {
		return "", fmt.Errorf("autodiscover: invalid email")
	}
	domain := email[strings.LastIndex(email, "@")+1:]
	if cfg.Tenants != nil {
		if _, err := cfg.Tenants.ResolveByDomain(domain); err != nil {
			return "", fmt.Errorf("autodiscover: unknown domain %q", domain)
		}
	}
	host := cfg.BridgeHost
	if host == "" {
		host = "mail-bridge.local"
	}
	port := cfg.HTTPPort
	if port == 0 {
		port = 8151
	}
	ssl := "off"
	if cfg.UseTLS {
		ssl = "on"
	}
	var b strings.Builder
	if err := tmpl.Execute(&b, map[string]any{
		"Email":    email,
		"IMAPHost": host,
		"IMAPPort": 993,
		"IMAPSSL":  ssl,
		"EWSHost":  host,
		"EXCHSSL":  ssl,
		"HTTPPort": port,
	}); err != nil {
		return "", err
	}
	return b.String(), nil
}
