// Package autodiscover — генерация Autodiscover XML для Outlook/мобильных (AC-C3).
package autodiscover

import (
	"fmt"
	"strings"
	"text/template"

	"era/services/platform/tenant"
)

// Config — параметры Autodiscover-ответа.
type Config struct {
	Email      string
	Tenants    *tenant.Store
	MailHost   string
	IMAPHost   string
	SMTPHost   string
	EWSHost    string
	CalDAVHost string
	IMAPPort   int
	SMTPPort   int
	HTTPPort   int
	SMTPUseTLS bool
	UseTLS     bool // IMAP/EXCH SSL on when true (R2-C)
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
        <DomainRequired>off</DomainRequired>
        <LoginName>{{ .Email }}</LoginName>
        <SPA>off</SPA>
        <SSL>{{ .IMAPSSL }}</SSL>
        <Server>{{ .SMTPHost }}</Server>
        <Port>{{ .SMTPPort }}</Port>
        <DomainRequired>off</DomainRequired>
        <LoginName>{{ .Email }}</LoginName>
        <SPA>off</SPA>
        <SSL>{{ .SMTPTLS }}</SSL>
        <AuthRequired>on</AuthRequired>
        <UsePOPAuth>on</UsePOPAuth>
      </Protocol>
      <Protocol>
        <Type>EXCH</Type>
        <Server>{{ .EWSHost }}</Server>
        <EwsUrl>https://{{ .EWSHost }}:{{ .HTTPPort }}/ews/Exchange.asmx</EwsUrl>
        <DomainRequired>off</DomainRequired>
        <LoginName>{{ .Email }}</LoginName>
        <SPA>off</SPA>
        <SSL>{{ .EXCHSSL }}</SSL>
        <Server>{{ .CalDAVHost }}</Server>
        <Url>https://{{ .CalDAVHost }}:{{ .HTTPPort }}/caldav/{{ .Email }}/</Url>
        <DomainRequired>off</DomainRequired>
        <LoginName>{{ .Email }}</LoginName>
      </Protocol>
    </Account>
  </Response>
</Autodiscover>
`

var tmpl = template.Must(template.New("autodiscover").Parse(autodiscoverTmpl))

// Render строит Autodiscover XML для email; tenant резолвится по домену.
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
	smtpTLS := "off"
	if cfg.SMTPUseTLS {
		smtpTLS = "on"
	}
	imapSSL := "off"
	exchSSL := "off"
	if cfg.UseTLS || cfg.SMTPUseTLS {
		imapSSL = "on"
		exchSSL = "on"
	}
	var b strings.Builder
	err := tmpl.Execute(&b, map[string]any{
		"Email":      email,
		"IMAPHost":   cfg.IMAPHost,
		"IMAPPort":   cfg.IMAPPort,
		"IMAPSSL":    imapSSL,
		"SMTPHost":   cfg.SMTPHost,
		"SMTPPort":   cfg.SMTPPort,
		"SMTPTLS":    smtpTLS,
		"EWSHost":    orDefault(cfg.EWSHost, cfg.MailHost),
		"EXCHSSL":    exchSSL,
		"CalDAVHost": orDefault(cfg.CalDAVHost, cfg.MailHost),
		"HTTPPort":   orPort(cfg.HTTPPort, 8150),
	})
	if err != nil {
		return "", err
	}
	return b.String(), nil
}

func orDefault(v, def string) string {
	if v != "" {
		return v
	}
	return def
}

func orPort(p, def int) int {
	if p > 0 {
		return p
	}
	return def
}
